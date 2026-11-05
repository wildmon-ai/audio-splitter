package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	// Durations above this come from damaged headers, and would plan an
	// absurd number of clips.
	maxDurationSec = 31 * 24 * 3600

	// A complete one-minute clip is at least this large at any common bit
	// rate, so only smaller files need the slower duration check.
	quickCompleteBytes = 256 * 1024
)

type tools struct {
	ffmpeg, ffprobe string
}

// findTools locates ffmpeg and ffprobe: next to this executable (packaged
// app), in the repository's resources/<arch>/ (development, see
// scripts/download-ffmpeg.sh), or else on PATH.
func findTools() tools {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		resources := filepath.Join(dir, "..", "resources")
		for _, d := range []string{dir, filepath.Join(resources, electronArch()), resources} {
			t := tools{ffmpeg: filepath.Join(d, "ffmpeg"+ext), ffprobe: filepath.Join(d, "ffprobe"+ext)}
			if isFile(t.ffmpeg) && isFile(t.ffprobe) {
				return t
			}
		}
	}
	return tools{ffmpeg: "ffmpeg" + ext, ffprobe: "ffprobe" + ext}
}

// electronArch names this CPU the way electron-builder and resources/ do.
func electronArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "arm":
		return "armv7l"
	default:
		return runtime.GOARCH
	}
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// check fails early when ffmpeg or ffprobe cannot run at all, instead of
// reporting the same error for every file.
func (t tools) check() error {
	for _, bin := range []string{t.ffmpeg, t.ffprobe} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s not found: it should come with the app, or be installed on this computer", filepath.Base(bin))
		}
	}
	return nil
}

type mediaInfo struct {
	duration float64 // seconds
	codec    string  // first audio stream's codec
}

func probe(ctx context.Context, ffprobe, path string) (mediaInfo, error) {
	out, err := runTool(ctx, ffprobe,
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "format=duration:stream=codec_name",
		"-of", "json",
		path,
	)
	if err != nil {
		return mediaInfo{}, err
	}
	var parsed struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return mediaInfo{}, fmt.Errorf("unexpected ffprobe output: %w", err)
	}
	if len(parsed.Streams) == 0 {
		return mediaInfo{}, errors.New("no audio stream found")
	}
	duration, err := strconv.ParseFloat(parsed.Format.Duration, 64)
	switch {
	case err != nil || math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0:
		return mediaInfo{}, errors.New("could not read the duration")
	case duration > maxDurationSec:
		return mediaInfo{}, fmt.Errorf("reported duration of %.0f hours looks wrong; the file may be damaged", duration/3600)
	}
	return mediaInfo{duration: duration, codec: parsed.Streams[0].CodecName}, nil
}

// cutClip writes one clip. ffmpeg writes to a hidden temporary file that is
// renamed into place only when it succeeds, so a crash, a cancel or a failure
// never leaves a partial clip that a later run would take for a finished one.
func cutClip(ctx context.Context, ffmpeg string, t clipTask) error {
	if err := os.MkdirAll(filepath.Dir(t.output), 0o755); err != nil {
		return err
	}
	tmp := partialPath(t.output)
	if _, err := runTool(ctx, ffmpeg, clipArgs(t, tmp)...); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, t.output); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func clipArgs(t clipTask, output string) []string {
	args := []string{
		"-nostdin", "-hide_banner", "-v", "error",
		"-threads", "1",
		// -ss before -i seeks in the input. After -i, ffmpeg decodes and
		// throws away everything before the clip, which gets slower the
		// later the clip starts in a long recording.
		"-ss", strconv.Itoa(t.offset),
		"-i", t.input,
		"-t", strconv.Itoa(clipSeconds),
		"-map", "0:a:0", "-ac", "1",
	}
	if codec := keepCodec(t.codec); codec != "" {
		args = append(args, "-c:a", codec)
	}
	return append(args, "-y", output)
}

// keepCodec returns the encoder that keeps the source's sample format, or ""
// to let ffmpeg pick one from the file extension. Without it, 24- and 32-bit
// WAV recordings would be written as 16-bit, ffmpeg's default for .wav.
func keepCodec(codec string) string {
	if strings.HasPrefix(codec, "pcm_") {
		return codec
	}
	return ""
}

// partialPath is where ffmpeg writes a clip until it is complete. It keeps the
// clip's extension, which ffmpeg uses to choose the output format, and starts
// with a dot so scans skip it.
func partialPath(output string) string {
	dir, name := filepath.Split(output)
	return filepath.Join(dir, "."+name+".partial"+filepath.Ext(name))
}

// existingClip reports whether output exists and whether it is a complete
// clip. Small files are probed: earlier versions could leave clips of a few
// milliseconds, and those are cut again instead of being skipped forever.
func existingClip(ctx context.Context, ffprobe, output string) (exists, complete bool) {
	info, err := os.Stat(output)
	if err != nil {
		return false, false
	}
	if info.Size() >= quickCompleteBytes {
		return true, true
	}
	m, err := probe(ctx, ffprobe, output)
	return true, err == nil && m.duration >= minClipSeconds
}

// runTool runs ffmpeg or ffprobe and returns its stdout.
func runTool(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, toolError(err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// toolError prefers the tool's last error line ("Invalid data found when
// processing input") to the bare exit status, and falls back to err when the
// tool printed nothing, e.g. because it could not be started.
func toolError(err error, stderr string) error {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
		return errors.New(last)
	}
	return err
}
