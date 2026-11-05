package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

type event struct {
	Type    string `json:"type"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Phase   string `json:"phase"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
}

// runJob runs one job and returns its done event and the events it sent.
func runJob(t *testing.T, ctx context.Context, cfg Config, bins tools) (doneEvent, []event) {
	t.Helper()
	var buf bytes.Buffer
	done := run(ctx, cfg, bins, newEmitter(&buf))
	var events []event
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var e event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("bad event line %q: %v", line, err)
		}
		events = append(events, e)
	}
	return done, events
}

func logs(events []event, level string) []string {
	var messages []string
	for _, e := range events {
		if e.Type == "log" && (level == "" || e.Level == level) {
			messages = append(messages, e.Message)
		}
	}
	return messages
}

func assertMinuteLong(t *testing.T, bins tools, path string) {
	t.Helper()
	if d := probeOrFail(t, bins, path).duration; math.Abs(d-60) > 0.01 {
		t.Errorf("%s lasts %.3f s, want 60 s", filepath.Base(path), d)
	}
}

func TestRunConsecutiveFiveMinuteRecordings(t *testing.T) {
	// End-to-end version of the reported bug: consecutive recordings of
	// 300.032 s, one clip every 5 minutes. Each recording must give one full
	// clip, with no extra near-empty clip taking the next recording's name.
	bins := requireFFmpeg(t)
	root := t.TempDir()
	src, out := filepath.Join(root, "src"), filepath.Join(root, "out")
	names := []string{"20260119_144815.WAV", "20260119_145315.WAV", "20260119_145815.WAV"}
	for _, n := range names {
		makeRecording(t, filepath.Join(src, "site", n), 300.032, "")
	}
	cfg := Config{SourceDir: src, OutputDir: out, RecordStep: 1, TargetStep: 5, Threads: 4}

	done, events := runJob(t, context.Background(), cfg, bins)
	if want := (doneEvent{Type: "done", TotalClips: 3}); done != want {
		t.Fatalf("done = %+v, want %+v\nlog: %v", done, want, logs(events, ""))
	}
	if got := fileNames(t, filepath.Join(out, "site")); !slices.Equal(got, names) {
		t.Errorf("output = %v, want %v", got, names)
	}
	for _, n := range names {
		assertMinuteLong(t, bins, filepath.Join(out, "site", n))
	}

	// Running again resumes: everything already exists.
	done, _ = runJob(t, context.Background(), cfg, bins)
	if want := (doneEvent{Type: "done", Skipped: 3}); done != want {
		t.Errorf("second run: done = %+v, want %+v", done, want)
	}
}

func TestRunRecreatesNearEmptyClipsFromEarlierVersions(t *testing.T) {
	bins := requireFFmpeg(t)
	root := t.TempDir()
	src, out := filepath.Join(root, "src"), filepath.Join(root, "out")
	makeRecording(t, filepath.Join(src, "20260119_144815.WAV"), 300.032, "")
	makeRecording(t, filepath.Join(src, "20260119_145315.WAV"), 300.032, "")
	// The 32 ms clip v0.1.0 wrote under the second recording's name.
	makeRecording(t, filepath.Join(out, "20260119_145315.WAV"), 0.032, "")

	done, events := runJob(t, context.Background(), Config{SourceDir: src, OutputDir: out, RecordStep: 1, TargetStep: 5}, bins)
	if want := (doneEvent{Type: "done", TotalClips: 2}); done != want {
		t.Fatalf("done = %+v, want %+v\nlog: %v", done, want, logs(events, ""))
	}
	assertMinuteLong(t, bins, filepath.Join(out, "20260119_145315.WAV"))
	if warnings := logs(events, "warn"); len(warnings) != 1 || !strings.Contains(warnings[0], "Re-creating") {
		t.Errorf("warnings = %v, want one about re-creating the clip", warnings)
	}
}

func TestRunDoesNotScanAnOutputFolderInsideTheSource(t *testing.T) {
	bins := requireFFmpeg(t)
	src := t.TempDir()
	out := filepath.Join(src, "clips")
	makeRecording(t, filepath.Join(src, "rec.wav"), 130, "")
	cfg := Config{SourceDir: src, OutputDir: out, RecordStep: 1, TargetStep: 1}

	if done, events := runJob(t, context.Background(), cfg, bins); done != (doneEvent{Type: "done", TotalClips: 2}) {
		t.Fatalf("first run: done = %+v\nlog: %v", done, logs(events, ""))
	}
	// Before, the second run cut clips out of the first run's clips.
	if done, events := runJob(t, context.Background(), cfg, bins); done != (doneEvent{Type: "done", Skipped: 2}) {
		t.Fatalf("second run: done = %+v\nlog: %v", done, logs(events, ""))
	}
	if got := fileNames(t, out); !slices.Equal(got, []string{"rec_0000m.wav", "rec_0001m.wav"}) {
		t.Errorf("output = %v", got)
	}
}

func TestRunReportsProblemsInsteadOfFinishingSilently(t *testing.T) {
	bins := requireFFmpeg(t)
	root := t.TempDir()
	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		cfg  Config
		want string
	}{
		"no audio files":     {Config{SourceDir: empty, OutputDir: filepath.Join(root, "out"), TargetStep: 1}, "no audio files"},
		"output is source":   {Config{SourceDir: empty, OutputDir: empty, TargetStep: 1}, "different from the source"},
		"missing source":     {Config{SourceDir: filepath.Join(root, "nope"), OutputDir: filepath.Join(root, "out"), TargetStep: 1}, "not found"},
		"ffmpeg not bundled": {Config{SourceDir: empty, OutputDir: filepath.Join(root, "out"), TargetStep: 1}, "not found"},
	} {
		t.Run(name, func(t *testing.T) {
			b := bins
			if name == "ffmpeg not bundled" {
				b = tools{ffmpeg: filepath.Join(root, "no-ffmpeg"), ffprobe: filepath.Join(root, "no-ffprobe")}
			}
			done, events := runJob(t, context.Background(), tc.cfg, b)
			if !strings.Contains(strings.ToLower(done.Error), tc.want) {
				t.Errorf("done.Error = %q, want it to mention %q", done.Error, tc.want)
			}
			if errs := logs(events, "error"); len(errs) != 1 {
				t.Errorf("error logs = %v, want the same message once", errs)
			}
		})
	}
}

func TestRunStopsAtOnceWhenAToolCannotStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell scripts in place of ffmpeg and ffprobe")
	}
	// v0.1.0 for Apple Silicon shipped Homebrew's ffprobe, which macOS stops
	// at launch on Macs without Homebrew's libraries. Every file failed with
	// "signal: abort trap", and the summary said "0 errors".
	for name, tc := range map[string]struct {
		ffprobe []string
		want    string
	}{
		"missing library": {
			[]string{
				`echo 'dyld[1]: Library not loaded: /opt/homebrew/lib/libavdevice.62.dylib' >&2`,
				`echo "  Reason: tried: '/opt/homebrew/lib/libavdevice.62.dylib' (no such file)" >&2`,
				"exit 134",
			},
			"Could not start ffprobe (dyld[1]: Library not loaded: /opt/homebrew/lib/libavdevice.62.dylib)",
		},
		"killed without a message": {[]string{"kill -KILL $$"}, "Could not start ffprobe (signal: killed)"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			touch(t, root, "src/20250510_151000.WAV")
			bins := tools{
				ffmpeg:  writeScript(t, root, "ffmpeg", "exit 0"),
				ffprobe: writeScript(t, root, "ffprobe", tc.ffprobe...),
			}
			cfg := Config{SourceDir: filepath.Join(root, "src"), OutputDir: filepath.Join(root, "out"), TargetStep: 1}

			done, events := runJob(t, context.Background(), cfg, bins)
			if !strings.HasPrefix(done.Error, tc.want) {
				t.Errorf("done.Error = %q, want it to start with %q", done.Error, tc.want)
			}
			if errs := logs(events, "error"); !slices.Equal(errs, []string{done.Error}) {
				t.Errorf("error logs = %q, want only the start failure", errs)
			}
		})
	}
}

// writeScript writes an executable shell script that stands in for a tool.
func writeScript(t *testing.T, dir, name string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+strings.Join(lines, "\n")+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunCountsUnreadableFilesAsErrors(t *testing.T) {
	bins := requireFFmpeg(t)
	root := t.TempDir()
	src := filepath.Join(root, "src")
	makeRecording(t, filepath.Join(src, "good.wav"), 60, "")
	if err := os.WriteFile(filepath.Join(src, "broken.wav"), []byte("not audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	done, events := runJob(t, context.Background(), Config{SourceDir: src, OutputDir: filepath.Join(root, "out"), TargetStep: 1}, bins)
	// Before, the summary said "0 errors" when a file could not be read.
	if want := (doneEvent{Type: "done", TotalClips: 1, Errors: 1}); done != want {
		t.Errorf("done = %+v, want %+v\nlog: %v", done, want, logs(events, ""))
	}
}

func TestRunWarnsAboutUnreadableFoldersAndGoesOn(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions and a non-root user")
	}
	bins := requireFFmpeg(t)
	root := t.TempDir()
	src := filepath.Join(root, "src")
	makeRecording(t, filepath.Join(src, "rec.wav"), 60, "")
	locked := filepath.Join(src, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	done, events := runJob(t, context.Background(), Config{SourceDir: src, OutputDir: filepath.Join(root, "out"), TargetStep: 1}, bins)
	if done != (doneEvent{Type: "done", TotalClips: 1}) {
		t.Errorf("done = %+v\nlog: %v", done, logs(events, ""))
	}
	if warnings := logs(events, "warn"); len(warnings) != 1 || !strings.Contains(warnings[0], "locked") {
		t.Errorf("warnings = %v, want one about the locked folder", warnings)
	}
}

func TestRunCancelled(t *testing.T) {
	bins := requireFFmpeg(t)
	root := t.TempDir()
	src, out := filepath.Join(root, "src"), filepath.Join(root, "out")
	makeRecording(t, filepath.Join(src, "rec.wav"), 300, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done, _ := runJob(t, ctx, Config{SourceDir: src, OutputDir: out, TargetStep: 1}, bins)
	if !done.Cancelled || done.TotalClips != 0 || done.Errors != 0 {
		t.Errorf("done = %+v, want cancelled with nothing extracted and no errors", done)
	}
}

func TestRunReportsProgressPhases(t *testing.T) {
	bins := requireFFmpeg(t)
	root := t.TempDir()
	src := filepath.Join(root, "src")
	makeRecording(t, filepath.Join(src, "rec.wav"), 130, "")
	_, events := runJob(t, context.Background(), Config{SourceDir: src, OutputDir: filepath.Join(root, "out"), TargetStep: 1}, bins)

	last := map[string]event{}
	for _, e := range events {
		if e.Type == "progress" {
			if prev, ok := last[e.Phase]; ok && e.Current < prev.Current {
				t.Errorf("%s progress went backwards: %d after %d", e.Phase, e.Current, prev.Current)
			}
			last[e.Phase] = e
		}
	}
	if s := last["scan"]; s.Current != 1 || s.Total != 1 {
		t.Errorf("final scan progress = %+v, want 1/1", s)
	}
	if c := last["clips"]; c.Current != 2 || c.Total != 2 {
		t.Errorf("final clip progress = %+v, want 2/2", c)
	}
}

func TestJobRunnerAnswersAnInvalidStartWithDone(t *testing.T) {
	// The app waits for a done event after every start; a bad config must
	// still produce one, or the UI would stay on "Processing…".
	var buf bytes.Buffer
	jobs := &jobRunner{events: newEmitter(&buf)}
	jobs.start(json.RawMessage(`{"source_dir":"/s","output_dir":"/o","target_step":0}`))
	jobs.shutdown()

	var done doneEvent
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &done); err != nil || done.Type != "done" || done.Error == "" {
		t.Errorf("last event = %s, want done with an error", lines[len(lines)-1])
	}
}
