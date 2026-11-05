package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// progressInterval limits progress events, so a resumed job with thousands of
// existing clips does not flood the app with one message per clip.
const progressInterval = 150 * time.Millisecond

// run executes one job and returns its done event.
func run(ctx context.Context, cfg Config, bins tools, events *emitter) doneEvent {
	started := time.Now()

	dirs, err := prepareFolders(cfg.SourceDir, cfg.OutputDir)
	if err != nil {
		return events.fail(err)
	}
	if err := bins.check(ctx); err != nil {
		if ctx.Err() != nil {
			return finished(events, started, 0, 0, 0, true)
		}
		return events.fail(err)
	}
	workers := workerCount(cfg.Threads)
	if cfg.Threads > workers {
		events.log(levelInfo, fmt.Sprintf("Using %d parallel processes, the most this computer can use well.", workers))
	}

	skipDir := ""
	if dirs.outputInsideSource {
		skipDir = dirs.output
		events.log(levelWarn, "The output folder is inside the source folder; it will not be scanned for recordings.")
	}
	files, err := collectFiles(dirs.source, skipDir, func(msg string) { events.log(levelWarn, msg) })
	if err != nil {
		return events.fail(fmt.Errorf("cannot read the source folder: %w", shortError(err)))
	}
	if len(files) == 0 {
		return events.fail(errors.New("no audio files (WAV, FLAC, MP3, OGG, M4A or OPUS) found in the source folder"))
	}

	events.log(levelInfo, fmt.Sprintf("Scanning %d files…", len(files)))
	tasks, probeErrors := planAll(ctx, bins.ffprobe, files, dirs, cfg, workers, events)
	if ctx.Err() != nil {
		return finished(events, started, 0, 0, probeErrors, true)
	}

	tasks, duplicates := dropDuplicateOutputs(tasks)
	for _, d := range duplicates {
		events.log(levelWarn, fmt.Sprintf("Skipped a clip of %s: %s is already planned from %s (overlapping recordings?).",
			relPath(dirs.source, d.task.input), relPath(dirs.output, d.task.output), relPath(dirs.source, d.keptFrom)))
	}
	if len(tasks) == 0 {
		events.log(levelWarn, "No clips to extract: the recordings are shorter than one minute or could not be read.")
		return finished(events, started, 0, 0, probeErrors, false)
	}

	events.log(levelInfo, fmt.Sprintf("%d clips planned across %d files — starting…", len(tasks), len(files)))
	extracted, skipped, cutErrors := processClips(ctx, bins, tasks, dirs, workers, events)
	return finished(events, started, extracted, skipped, probeErrors+cutErrors, ctx.Err() != nil)
}

// workerCount turns the Parallel Processes setting into a worker count. 0 means
// one per CPU core; large values are capped because each worker runs ffmpeg.
func workerCount(threads int) int {
	limit := 4 * runtime.NumCPU()
	switch {
	case threads <= 0:
		return runtime.NumCPU()
	case threads > limit:
		return limit
	}
	return threads
}

// planAll reads every file's duration in parallel and returns the clips in
// scan order, so the plan does not depend on which ffprobe finishes first.
func planAll(ctx context.Context, ffprobe string, files []string, dirs folders, cfg Config, workers int, events *emitter) ([]clipTask, int) {
	plans := make([][]clipTask, len(files))
	errs := make([]error, len(files))
	var probed atomic.Int64
	stopProgress := startProgress(events, "scan", len(files), &probed)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
loop:
	for i, path := range files {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer probed.Add(1)
			info, err := probe(ctx, ffprobe, path)
			if err != nil {
				errs[i] = err
				return
			}
			plans[i] = planClips(path, info.duration, info.codec, dirs.source, dirs.output, cfg.RecordStep, cfg.TargetStep)
		}()
	}
	wg.Wait()
	stopProgress()

	var tasks []clipTask
	failures := 0
	for i, path := range files {
		if errs[i] != nil && !errors.Is(errs[i], context.Canceled) {
			failures++
			events.log(levelError, fmt.Sprintf("%s: %v", relPath(dirs.source, path), errs[i]))
		}
		tasks = append(tasks, plans[i]...)
	}
	return tasks, failures
}

// processClips cuts the clips with a pool of workers. Clips that already
// exist complete are skipped, which is what makes a job resumable.
func processClips(ctx context.Context, bins tools, tasks []clipTask, dirs folders, workers int, events *emitter) (extracted, skipped, failed int) {
	var processed, nExtracted, nSkipped, nFailed atomic.Int64
	stopProgress := startProgress(events, "clips", len(tasks), &processed)

	queue := make(chan clipTask)
	go func() {
		defer close(queue)
		for _, t := range tasks {
			select {
			case queue <- t:
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range queue {
				if ctx.Err() != nil {
					return
				}
				exists, complete := existingClip(ctx, bins.ffprobe, t.output)
				if exists && complete {
					nSkipped.Add(1)
					processed.Add(1)
					continue
				}
				if exists {
					events.log(levelWarn, "Re-creating incomplete clip "+relPath(dirs.output, t.output))
				}
				err := cutClip(ctx, bins.ffmpeg, t)
				switch {
				case err == nil:
					nExtracted.Add(1)
				case ctx.Err() != nil:
					// Cancelled: the clip was not written, and that is not an error.
				default:
					nFailed.Add(1)
					events.log(levelError, fmt.Sprintf("%s: %v", relPath(dirs.output, t.output), err))
				}
				processed.Add(1)
			}
		}()
	}
	wg.Wait()
	stopProgress()
	return int(nExtracted.Load()), int(nSkipped.Load()), int(nFailed.Load())
}

// startProgress reports a phase's done/total count every progressInterval
// while it changes. A single goroutine sends the events, so the count shown in
// the app never goes backwards. stop sends the final count.
func startProgress(events *emitter, phase string, total int, done *atomic.Int64) (stop func()) {
	quit := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(progressInterval)
		defer ticker.Stop()
		last := int64(-1)
		report := func() {
			if n := done.Load(); n != last {
				last = n
				events.send(progressEvent{Type: "progress", Phase: phase, Current: int(n), Total: total})
			}
		}
		report()
		for {
			select {
			case <-ticker.C:
				report()
			case <-quit:
				report()
				return
			}
		}
	}()
	return func() {
		close(quit)
		<-stopped
	}
}

func finished(events *emitter, started time.Time, extracted, skipped, failed int, cancelled bool) doneEvent {
	verb := "Finished"
	if cancelled {
		verb = "Cancelled"
	}
	elapsed := time.Since(started).Round(time.Second)
	events.log(levelInfo, fmt.Sprintf("%s in %dm %02ds — %d clips extracted, %d skipped (already existed), %d errors",
		verb, int(elapsed.Minutes()), int(elapsed.Seconds())%60, extracted, skipped, failed))
	return doneEvent{Type: "done", TotalClips: extracted, Skipped: skipped, Errors: failed, Cancelled: cancelled}
}
