package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestClipArgsSeeksBeforeOpeningTheInput(t *testing.T) {
	args := clipArgs(clipTask{input: "in.wav", offset: 7140, codec: "pcm_s24le"}, "out.wav")
	ss, in := slices.Index(args, "-ss"), slices.Index(args, "-i")
	if ss < 0 || in < 0 || ss > in || args[ss+1] != "7140" {
		t.Errorf("want -ss 7140 before -i: %v", args)
	}
	if i := slices.Index(args, "-c:a"); i < 0 || args[i+1] != "pcm_s24le" {
		t.Errorf("want -c:a pcm_s24le to keep the bit depth: %v", args)
	}
	if args[len(args)-1] != "out.wav" {
		t.Errorf("output must be the last argument: %v", args)
	}
}

func TestKeepCodec(t *testing.T) {
	for codec, want := range map[string]string{
		"pcm_s16le": "pcm_s16le",
		"pcm_s24le": "pcm_s24le",
		"pcm_f32le": "pcm_f32le",
		"flac":      "", // ffmpeg already keeps FLAC's bit depth
		"mp3":       "",
		"":          "",
	} {
		if got := keepCodec(codec); got != want {
			t.Errorf("keepCodec(%q) = %q, want %q", codec, got, want)
		}
	}
}

func TestPartialPath(t *testing.T) {
	got := partialPath(filepath.Join("out", "site", "x.WAV"))
	if want := filepath.Join("out", "site", ".x.WAV.partial.WAV"); got != want {
		t.Errorf("partialPath() = %q, want %q", got, want)
	}
}

func TestToolError(t *testing.T) {
	base := errors.New("exit status 1")
	if got := toolError(base, " \n"); got != base {
		t.Errorf("empty stderr: got %v, want the original error", got)
	}
	got := toolError(base, "first line\nInvalid data found when processing input\n")
	if got.Error() != "Invalid data found when processing input" {
		t.Errorf("got %q, want the last stderr line", got)
	}
}

// requireFFmpeg skips tests that need the real ffmpeg when it is not installed.
func requireFFmpeg(t *testing.T) tools {
	t.Helper()
	bins := tools{ffmpeg: "ffmpeg", ffprobe: "ffprobe"}
	if err := bins.check(); err != nil {
		t.Skip("ffmpeg/ffprobe not installed")
	}
	return bins
}

// makeRecording writes a mono 8 kHz sine wave of the given length. codec ""
// lets ffmpeg pick the default for the extension.
func makeRecording(t *testing.T, path string, seconds float64, codec string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	args := []string{"-v", "error", "-y", "-f", "lavfi", "-i", fmt.Sprintf("sine=f=440:r=8000:d=%g", seconds)}
	if codec != "" {
		args = append(args, "-c:a", codec)
	}
	if out, err := exec.Command("ffmpeg", append(args, path)...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v\n%s", err, out)
	}
}

func probeOrFail(t *testing.T, bins tools, path string) mediaInfo {
	t.Helper()
	m, err := probe(context.Background(), bins.ffprobe, path)
	if err != nil {
		t.Fatalf("probe %s: %v", path, err)
	}
	return m
}

func fileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestCutClipKeeps24BitAudio(t *testing.T) {
	bins := requireFFmpeg(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.wav")
	makeRecording(t, in, 130, "pcm_s24le")

	task := clipTask{input: in, output: filepath.Join(dir, "out", "clip.wav"), offset: 60, codec: probeOrFail(t, bins, in).codec}
	if err := cutClip(context.Background(), bins.ffmpeg, task); err != nil {
		t.Fatal(err)
	}
	got := probeOrFail(t, bins, task.output)
	if got.codec != "pcm_s24le" || math.Abs(got.duration-60) > 0.01 {
		t.Errorf("clip = %+v, want 60 s of pcm_s24le", got)
	}
	if names := fileNames(t, filepath.Join(dir, "out")); !slices.Equal(names, []string{"clip.wav"}) {
		t.Errorf("output folder has %v, want only clip.wav", names)
	}
}

func TestCutClipFailureLeavesNothingBehind(t *testing.T) {
	bins := requireFFmpeg(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "broken.wav")
	if err := os.WriteFile(in, []byte("not audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := cutClip(context.Background(), bins.ffmpeg, clipTask{input: in, output: filepath.Join(dir, "out", "clip.wav")})
	if err == nil || strings.TrimSpace(err.Error()) == "" {
		t.Fatalf("err = %v, want a readable error", err)
	}
	if names := fileNames(t, filepath.Join(dir, "out")); len(names) != 0 {
		t.Errorf("failed cut left %v behind", names)
	}
}

func TestCutClipExplainsWhyFFmpegDidNotStart(t *testing.T) {
	// The old code printed an empty message ("✗ clip.wav: ") in this case.
	dir := t.TempDir()
	notExecutable := filepath.Join(dir, "ffmpeg")
	if err := os.WriteFile(notExecutable, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	err := cutClip(context.Background(), notExecutable, clipTask{input: "in.wav", output: filepath.Join(dir, "clip.wav")})
	if err == nil || strings.TrimSpace(err.Error()) == "" {
		t.Errorf("err = %v, want a readable error", err)
	}
}

func TestExistingClip(t *testing.T) {
	bins := requireFFmpeg(t)
	dir := t.TempDir()
	full := filepath.Join(dir, "full.wav")
	makeRecording(t, full, 60, "") // large: accepted from its size
	small := filepath.Join(dir, "small.flac")
	makeRecording(t, small, 60, "flac") // small but complete: accepted after probing
	tiny := filepath.Join(dir, "tiny.wav")
	makeRecording(t, tiny, 0.032, "") // what earlier versions left at the end of a recording

	for _, tc := range []struct {
		path             string
		exists, complete bool
	}{
		{full, true, true},
		{small, true, true},
		{tiny, true, false},
		{filepath.Join(dir, "missing.wav"), false, false},
	} {
		exists, complete := existingClip(context.Background(), bins.ffprobe, tc.path)
		if exists != tc.exists || complete != tc.complete {
			t.Errorf("existingClip(%s) = %v, %v; want %v, %v", filepath.Base(tc.path), exists, complete, tc.exists, tc.complete)
		}
	}
}
