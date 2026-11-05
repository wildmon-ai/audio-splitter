package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Limits shared with the UI (src/utils/validation.ts).
const (
	maxTargetStep = 100_000 // minutes between clips (~69 days)
	maxRecordStep = 1_000
	maxThreads    = 1_024
)

// Config is a validated job configuration.
type Config struct {
	SourceDir  string
	OutputDir  string
	RecordStep int // real-world minutes per recorded minute, for timestamped names
	TargetStep int // minutes between the start of consecutive clips
	Threads    int // parallel ffmpeg processes; 0 means one per CPU core
}

func parseConfig(raw json.RawMessage) (Config, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return Config{}, errors.New("missing job configuration")
	}
	var in struct {
		SourceDir  string   `json:"source_dir"`
		OutputDir  string   `json:"output_dir"`
		RecordStep *float64 `json:"record_step"`
		TargetStep *float64 `json:"target_step"`
		NumThreads *float64 `json:"num_threads"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return Config{}, fmt.Errorf("invalid job configuration: %w", err)
	}
	if in.TargetStep == nil {
		return Config{}, errors.New("extract Every N Minutes is missing")
	}

	cfg := Config{SourceDir: in.SourceDir, OutputDir: in.OutputDir}
	var err error
	if cfg.TargetStep, err = wholeNumber(in.TargetStep, 0, 1, maxTargetStep, "Extract Every N Minutes"); err != nil {
		return Config{}, err
	}
	if cfg.RecordStep, err = wholeNumber(in.RecordStep, 1, 1, maxRecordStep, "Record step"); err != nil {
		return Config{}, err
	}
	if cfg.Threads, err = wholeNumber(in.NumThreads, 0, 0, maxThreads, "Parallel Processes"); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// wholeNumber validates an optional JSON number. A step of 0 used to make the
// planner loop forever, so out-of-range values are rejected, not clamped.
func wholeNumber(v *float64, def, lo, hi int, name string) (int, error) {
	if v == nil {
		return def, nil
	}
	f := *v
	if f != math.Trunc(f) || f < float64(lo) || f > float64(hi) {
		return 0, fmt.Errorf("%s must be a whole number from %d to %d (got %v)", name, lo, hi, f)
	}
	return int(f), nil
}

// folders are a job's source and output folders, with symlinks resolved.
type folders struct {
	source, output     string
	outputInsideSource bool
}

// prepareFolders checks the source folder and creates the output folder.
func prepareFolders(sourceDir, outputDir string) (folders, error) {
	if strings.TrimSpace(sourceDir) == "" {
		return folders{}, errors.New("choose a source folder")
	}
	if strings.TrimSpace(outputDir) == "" {
		return folders{}, errors.New("choose an output folder")
	}
	src, err := filepath.Abs(sourceDir)
	if err != nil {
		return folders{}, err
	}
	info, err := os.Stat(src)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return folders{}, fmt.Errorf("source folder not found: %s", src)
	case err != nil:
		return folders{}, fmt.Errorf("cannot open the source folder: %w", err)
	case !info.IsDir():
		return folders{}, fmt.Errorf("the source is not a folder: %s", src)
	}
	out, err := filepath.Abs(outputDir)
	if err != nil {
		return folders{}, err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return folders{}, fmt.Errorf("cannot create the output folder: %w", err)
	}

	// Compare real paths, so a symlink cannot point the output at the source.
	src, out = realPath(src), realPath(out)
	if samePath(src, out) {
		return folders{}, errors.New("the output folder must be different from the source folder")
	}
	return folders{source: src, output: out, outputInsideSource: isInside(out, src)}, nil
}

func realPath(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return filepath.Clean(p)
}

// pathKey normalizes a path for comparison. Windows and macOS file systems are
// case-insensitive by default.
func pathKey(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.ToLower(p)
	}
	return p
}

func samePath(a, b string) bool {
	return pathKey(a) == pathKey(b)
}

// isInside reports whether child is strictly inside parent.
func isInside(child, parent string) bool {
	rel, err := filepath.Rel(pathKey(parent), pathKey(child))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// relPath shortens path for log messages.
func relPath(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}
