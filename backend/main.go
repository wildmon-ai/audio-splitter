package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const sampleDuration = 60

var audioExtensions = map[string]bool{
	".wav": true, ".mp3": true, ".flac": true,
	".ogg": true, ".m4a": true, ".opus": true,
}

var dateRe = regexp.MustCompile(`^(.*?)(\d{8}_\d{6})(.*)$`)

var emitMu sync.Mutex

func emit(obj any) {
	b, _ := json.Marshal(obj)
	emitMu.Lock()
	os.Stdout.Write(b)
	os.Stdout.Write([]byte{'\n'})
	emitMu.Unlock()
}

func ffmpegBins() (string, string) {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	base := filepath.Dir(os.Args[0])
	ffmpeg := filepath.Join(base, "ffmpeg"+ext)
	if _, err := os.Stat(ffmpeg); err == nil {
		return ffmpeg, filepath.Join(base, "ffprobe"+ext)
	}
	base = filepath.Join(filepath.Dir(os.Args[0]), "..", "resources")
	ffmpeg = filepath.Join(base, "ffmpeg"+ext)
	if _, err := os.Stat(ffmpeg); err == nil {
		return ffmpeg, filepath.Join(base, "ffprobe"+ext)
	}
	return "ffmpeg" + ext, "ffprobe" + ext
}

func getDuration(ffprobe, filePath string) (float64, error) {
	out, err := exec.Command(ffprobe,
		"-i", filePath,
		"-show_entries", "format=duration",
		"-v", "quiet",
		"-of", "csv=p=0",
	).Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

func cutClip(ffmpegBin, input, output string, offset int) error {
	var stderr bytes.Buffer
	cmd := exec.Command(ffmpegBin,
		"-threads", "1",
		"-i", input,
		"-ss", strconv.Itoa(offset),
		"-t", strconv.Itoa(sampleDuration),
		"-y", "-vn", "-ac", "1",
		output,
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		return fmt.Errorf("%s", lines[len(lines)-1])
	}
	return nil
}

type parsedDate struct {
	prefix string
	t      time.Time
	suffix string
}

func parseDateFromName(stem string) *parsedDate {
	m := dateRe.FindStringSubmatch(stem)
	if m == nil {
		return nil
	}
	t, err := time.Parse("20060102_150405", m[2])
	if err != nil {
		return nil
	}
	return &parsedDate{prefix: m[1], t: t, suffix: m[3]}
}

type clipTask struct {
	input    string
	output   string
	offset   int
	filename string
}

// planFile runs ffprobe and returns all clip tasks for one file.
func planFile(ffprobeBin, filePath, sourceDir, outputDir string, recordStep, targetStep int) ([]clipTask, error) {
	durSec, err := getDuration(ffprobeBin, filePath)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	stem := strings.TrimSuffix(filename, ext)
	parsed := parseDateFromName(stem)

	relDir, _ := filepath.Rel(sourceDir, filepath.Dir(filePath))
	outDir := filepath.Join(outputDir, relDir)
	os.MkdirAll(outDir, 0o755)

	audioStepSec := targetStep * 60
	timeAdvance := time.Duration(targetStep*recordStep) * time.Minute

	clipTime := time.Time{}
	if parsed != nil {
		clipTime = parsed.t
	}

	total := int(math.Ceil(durSec))
	offset := 0
	var tasks []clipTask

	for offset < total {
		var outName string
		if parsed != nil {
			outName = parsed.prefix + clipTime.Format("20060102_150405") + parsed.suffix + ext
		} else {
			outName = fmt.Sprintf("%s_%04dm%s", stem, offset/60, ext)
		}
		tasks = append(tasks, clipTask{
			input:    filePath,
			output:   filepath.Join(outDir, outName),
			offset:   offset,
			filename: filename,
		})
		offset += audioStepSec
		if parsed != nil {
			clipTime = clipTime.Add(timeAdvance)
		}
	}
	return tasks, nil
}

func collectFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if audioExtensions[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

var (
	cancelFunc context.CancelFunc
	cancelMu   sync.Mutex
)

func runProcessing(cfg map[string]any) {
	cancelMu.Lock()
	if cancelFunc != nil {
		cancelFunc()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancelFunc = cancel
	cancelMu.Unlock()
	startedAt := time.Now()

	sourceDir, _ := cfg["source_dir"].(string)
	outputDir, _ := cfg["output_dir"].(string)
	recordStep := intVal(cfg, "record_step", 1)
	targetStep := intVal(cfg, "target_step", 1)
	numWorkers := intVal(cfg, "num_threads", 0)
	if numWorkers < 1 {
		numWorkers = runtime.NumCPU()
	}

	ffmpegBin, ffprobeBin := ffmpegBins()

	files, err := collectFiles(sourceDir)
	if err != nil || len(files) == 0 {
		emit(map[string]any{"type": "done", "total_clips": 0, "skipped": 0, "errors": 0})
		return
	}

	emit(map[string]any{"type": "log", "level": "info",
		"message": fmt.Sprintf("Scanning %d files…", len(files))})

	// Phase 1: plan all clips in parallel (ffprobe only)
	type planResult struct {
		tasks    []clipTask
		filename string
		err      error
	}
	planCh := make(chan planResult, len(files))
	planSem := make(chan struct{}, numWorkers)
	var planWg sync.WaitGroup

	for _, f := range files {
		planWg.Add(1)
		planSem <- struct{}{}
		go func(path string) {
			defer planWg.Done()
			defer func() { <-planSem }()
			tasks, err := planFile(ffprobeBin, path, sourceDir, outputDir, recordStep, targetStep)
			planCh <- planResult{tasks: tasks, filename: filepath.Base(path), err: err}
		}(f)
	}
	planWg.Wait()
	close(planCh)

	var allTasks []clipTask
	for r := range planCh {
		if r.err != nil {
			emit(map[string]any{"type": "log", "level": "error",
				"message": fmt.Sprintf("✗ %s: %v", r.filename, r.err)})
			continue
		}
		allTasks = append(allTasks, r.tasks...)
	}

	totalClips := len(allTasks)
	if totalClips == 0 {
		emit(map[string]any{"type": "done", "total_clips": 0, "skipped": 0, "errors": 0})
		return
	}

	emit(map[string]any{"type": "log", "level": "info",
		"message": fmt.Sprintf("%d clips planned across %d files — starting…", totalClips, len(files))})
	emit(map[string]any{"type": "progress", "current": 0, "total": totalClips})

	// Phase 2: worker pool processes clips
	taskCh := make(chan clipTask, totalClips)
	for _, t := range allTasks {
		taskCh <- t
	}
	close(taskCh)

	var (
		doneClips   int64
		skipped     int64
		errors      int64
		workerWg    sync.WaitGroup
	)

	for i := 0; i < numWorkers; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for task := range taskCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if _, err := os.Stat(task.output); err == nil {
					atomic.AddInt64(&skipped, 1)
					emit(map[string]any{"type": "log", "level": "info",
						"message": fmt.Sprintf("⏭ skipped (already exists): %s", filepath.Base(task.output))})
				} else if err := cutClip(ffmpegBin, task.input, task.output, task.offset); err != nil {
					atomic.AddInt64(&errors, 1)
					emit(map[string]any{"type": "log", "level": "error",
						"message": fmt.Sprintf("✗ %s: %v", filepath.Base(task.output), err)})
				} else {
					atomic.AddInt64(&doneClips, 1)
				}

				done := atomic.LoadInt64(&doneClips) + atomic.LoadInt64(&skipped) + atomic.LoadInt64(&errors)
				emit(map[string]any{"type": "progress", "current": int(done), "total": totalClips})
			}
		}()
	}
	workerWg.Wait()

	elapsed := time.Since(startedAt).Round(time.Second)
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	emit(map[string]any{"type": "log", "level": "info",
		"message": fmt.Sprintf("Ended in %dm %ds — %d clips extracted, %d skipped, %d errors",
			mins, secs, doneClips, skipped, errors)})

	emit(map[string]any{
		"type":        "done",
		"total_clips": doneClips,
		"skipped":     skipped,
		"errors":      errors,
	})
}

func intVal(m map[string]any, key string, def int) int {
	v, ok := m[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return def
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var cmd map[string]any
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			continue
		}
		switch cmd["cmd"] {
		case "start":
			cfg, _ := cmd["config"].(map[string]any)
			go runProcessing(cfg)
		case "cancel":
			cancelMu.Lock()
			if cancelFunc != nil {
				cancelFunc()
			}
			cancelMu.Unlock()
		}
	}
}
