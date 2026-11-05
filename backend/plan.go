package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	clipSeconds = 60

	// A clip is only planned when at least this much audio is left. The
	// one-second slack keeps the last clip of a recording that stops a few
	// milliseconds short of a full minute. A recording that runs a few
	// milliseconds past its nominal length (300.032 s for a "5-minute" file)
	// does not get an extra near-empty clip, whose timestamped name would be
	// the same as the next recording's first clip.
	minClipSeconds = clipSeconds - 1

	timestampLayout = "20060102_150405"
)

var dateRe = regexp.MustCompile(`^(.*?)(\d{8}_\d{6})(.*)$`)

type clipTask struct {
	input  string // source recording
	output string // final clip path
	offset int    // clip start, in seconds from the start of the recording
	codec  string // source audio codec, as reported by ffprobe
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
	t, err := time.Parse(timestampLayout, m[2])
	if err != nil {
		return nil
	}
	return &parsedDate{prefix: m[1], t: t, suffix: m[3]}
}

// planClips lists the clips to cut from one recording of durSec seconds: one
// clip every targetStep minutes, as long as a full clip fits. Clips go to the
// same subfolder of outputDir as the recording has in sourceDir.
func planClips(input string, durSec float64, codec, sourceDir, outputDir string, recordStep, targetStep int) []clipTask {
	filename := filepath.Base(input)
	ext := filepath.Ext(filename)
	stem := strings.TrimSuffix(filename, ext)
	parsed := parseDateFromName(stem)

	outDir := outputDir
	if rel, err := filepath.Rel(sourceDir, filepath.Dir(input)); err == nil {
		outDir = filepath.Join(outputDir, rel)
	}

	var tasks []clipTask
	for offset := 0; float64(offset+minClipSeconds) <= durSec; offset += targetStep * 60 {
		name := fmt.Sprintf("%s_%04dm%s", stem, offset/60, ext)
		if parsed != nil {
			// The clip's real start time. Recorders with a record step above 1
			// advance their clock faster than the audio.
			start := parsed.t.Add(time.Duration(offset/60*recordStep) * time.Minute)
			name = parsed.prefix + start.Format(timestampLayout) + parsed.suffix + ext
		}
		tasks = append(tasks, clipTask{
			input:  input,
			output: filepath.Join(outDir, name),
			offset: offset,
			codec:  codec,
		})
	}
	return tasks
}

type duplicate struct {
	task     clipTask // the clip left out
	keptFrom string   // the recording whose clip has the same name
}

// dropDuplicateOutputs keeps the first task for each output path. Recordings
// that overlap in time can map two clips to the same timestamped name, and
// cutting both would make them overwrite each other.
func dropDuplicateOutputs(tasks []clipTask) (kept []clipTask, dropped []duplicate) {
	firstInput := make(map[string]string, len(tasks))
	for _, t := range tasks {
		key := pathKey(t.output)
		if input, ok := firstInput[key]; ok {
			dropped = append(dropped, duplicate{task: t, keptFrom: input})
			continue
		}
		firstInput[key] = t.input
		kept = append(kept, t)
	}
	return kept, dropped
}
