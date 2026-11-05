package main

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"
)

var (
	testSrc = filepath.Join(string(filepath.Separator)+"data", "src")
	testOut = filepath.Join(string(filepath.Separator)+"data", "out")
)

func outputNames(tasks []clipTask) []string {
	var names []string
	for _, t := range tasks {
		names = append(names, filepath.Base(t.output))
	}
	return names
}

func TestPlanClips(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		duration   float64
		recordStep int
		targetStep int
		want       []string
	}{
		{
			// The reported bug: a "5-minute" file of 300.032 s used to get a
			// second, 32 ms clip named like the next recording's first clip.
			name: "file a few ms over 5 minutes gets no extra clip",
			file: "20260119_144815.WAV", duration: 300.032, recordStep: 1, targetStep: 5,
			want: []string{"20260119_144815.WAV"},
		},
		{
			name: "every minute of a 300.032 s file",
			file: "20260119_144815.WAV", duration: 300.032, recordStep: 1, targetStep: 1,
			want: []string{"20260119_144815.WAV", "20260119_144915.WAV", "20260119_145015.WAV", "20260119_145115.WAV", "20260119_145215.WAV"},
		},
		{
			name: "last clip kept when the file is a few ms short",
			file: "20260119_153815.WAV", duration: 299.946667, recordStep: 1, targetStep: 1,
			want: []string{"20260119_153815.WAV", "20260119_153915.WAV", "20260119_154015.WAV", "20260119_154115.WAV", "20260119_154215.WAV"},
		},
		{
			name: "text around the timestamp is kept",
			file: "SITE01_20240601_060000_A.wav", duration: 3600, recordStep: 1, targetStep: 10,
			want: []string{
				"SITE01_20240601_060000_A.wav", "SITE01_20240601_061000_A.wav", "SITE01_20240601_062000_A.wav",
				"SITE01_20240601_063000_A.wav", "SITE01_20240601_064000_A.wav", "SITE01_20240601_065000_A.wav",
			},
		},
		{
			name: "record step scales the timestamps",
			file: "R_20240601_060000.wav", duration: 1800, recordStep: 2, targetStep: 10,
			want: []string{"R_20240601_060000.wav", "R_20240601_062000.wav", "R_20240601_064000.wav"},
		},
		{
			name: "timestamps cross midnight",
			file: "20240601_235500.flac", duration: 1200, recordStep: 1, targetStep: 10,
			want: []string{"20240601_235500.flac", "20240602_000500.flac"},
		},
		{
			name: "without a timestamp, clips are named by minute",
			file: "recording.mp3", duration: 1260, recordStep: 1, targetStep: 10,
			want: []string{"recording_0000m.mp3", "recording_0010m.mp3", "recording_0020m.mp3"},
		},
		{name: "59 s is enough for one clip", file: "short.wav", duration: 59, recordStep: 1, targetStep: 1, want: []string{"short_0000m.wav"}},
		{name: "under 59 s gives no clip", file: "short.wav", duration: 58.9, recordStep: 1, targetStep: 1, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := planClips(filepath.Join(testSrc, tc.file), tc.duration, "pcm_s16le", testSrc, testOut, tc.recordStep, tc.targetStep)
			if names := outputNames(got); !slices.Equal(names, tc.want) {
				t.Errorf("clips = %v, want %v", names, tc.want)
			}
		})
	}
}

func TestPlanClipsMirrorsSubfolders(t *testing.T) {
	input := filepath.Join(testSrc, "site", "day1", "rec.wav")
	got := planClips(input, 1300, "pcm_s24le", testSrc, testOut, 1, 10)
	dir := filepath.Join(testOut, "site", "day1")
	want := []clipTask{
		{input: input, output: filepath.Join(dir, "rec_0000m.wav"), offset: 0, codec: "pcm_s24le"},
		{input: input, output: filepath.Join(dir, "rec_0010m.wav"), offset: 600, codec: "pcm_s24le"},
		{input: input, output: filepath.Join(dir, "rec_0020m.wav"), offset: 1200, codec: "pcm_s24le"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("planClips() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestConsecutiveRecordingsDoNotShareClipNames(t *testing.T) {
	// Each 5-minute recording starts where the previous one ends.
	var tasks []clipTask
	for _, f := range []string{"20260119_144815.WAV", "20260119_145315.WAV", "20260119_145815.WAV"} {
		tasks = append(tasks, planClips(filepath.Join(testSrc, f), 300.032, "pcm_s16le", testSrc, testOut, 1, 5)...)
	}
	kept, dropped := dropDuplicateOutputs(tasks)
	if len(dropped) != 0 {
		t.Fatalf("dropped %d clips, want none: %+v", len(dropped), dropped)
	}
	for _, task := range kept {
		if filepath.Base(task.output) != filepath.Base(task.input) {
			t.Errorf("clip %s planned from %s; each recording should give only its own first clip", task.output, task.input)
		}
	}
}

func TestDropDuplicateOutputsKeepsTheFirst(t *testing.T) {
	a := clipTask{input: "a.wav", output: filepath.Join("out", "x.wav")}
	b := clipTask{input: "b.wav", output: filepath.Join("out", "x.wav")}
	c := clipTask{input: "b.wav", output: filepath.Join("out", "y.wav")}
	kept, dropped := dropDuplicateOutputs([]clipTask{a, b, c})
	if !reflect.DeepEqual(kept, []clipTask{a, c}) {
		t.Errorf("kept = %+v, want [a c]", kept)
	}
	if !reflect.DeepEqual(dropped, []duplicate{{task: b, keptFrom: "a.wav"}}) {
		t.Errorf("dropped = %+v, want b (kept from a.wav)", dropped)
	}
}

func TestParseDateFromName(t *testing.T) {
	p := parseDateFromName("SITE_20240601_060000_X")
	if p == nil || p.prefix != "SITE_" || p.suffix != "_X" || !p.t.Equal(time.Date(2024, 6, 1, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("parseDateFromName() = %+v", p)
	}
	for _, stem := range []string{"recording", "20241301_000000", "2024061_060000"} {
		if p := parseDateFromName(stem); p != nil {
			t.Errorf("parseDateFromName(%q) = %+v, want nil", stem, p)
		}
	}
}
