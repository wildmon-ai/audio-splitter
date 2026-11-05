package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	cfg, err := parseConfig(json.RawMessage(`{"source_dir":"/s","output_dir":"/o","record_step":2,"target_step":10,"num_threads":3}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Config{SourceDir: "/s", OutputDir: "/o", RecordStep: 2, TargetStep: 10, Threads: 3}); cfg != want {
		t.Errorf("parseConfig() = %+v, want %+v", cfg, want)
	}

	cfg, err = parseConfig(json.RawMessage(`{"source_dir":"/s","output_dir":"/o","target_step":5}`))
	if err != nil || cfg.RecordStep != 1 || cfg.Threads != 0 {
		t.Errorf("defaults: got %+v, %v; want record step 1 and automatic threads", cfg, err)
	}
}

func TestParseConfigRejectsInvalidValues(t *testing.T) {
	invalid := map[string]string{
		"target step 0 (an emptied input)": `{"target_step":0}`,
		"negative target step":             `{"target_step":-5}`,
		"fractional target step":           `{"target_step":2.5}`,
		"huge target step":                 `{"target_step":1e18}`,
		"missing target step":              `{"source_dir":"/s"}`,
		"record step 0":                    `{"target_step":5,"record_step":0}`,
		"negative threads":                 `{"target_step":5,"num_threads":-1}`,
		"number sent as text":              `{"target_step":"5"}`,
		"no config":                        ``,
		"null config":                      `null`,
	}
	for name, raw := range invalid {
		t.Run(name, func(t *testing.T) {
			if cfg, err := parseConfig(json.RawMessage(raw)); err == nil {
				t.Errorf("parseConfig(%s) = %+v, want an error", raw, cfg)
			}
		})
	}
}

func TestPrepareFolders(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("creates the output folder", func(t *testing.T) {
		out := filepath.Join(root, "new", "out")
		dirs, err := prepareFolders(src, out)
		if err != nil {
			t.Fatal(err)
		}
		if dirs.outputInsideSource || !isDir(out) {
			t.Errorf("got %+v; want a new output folder outside the source", dirs)
		}
	})
	t.Run("flags an output folder inside the source", func(t *testing.T) {
		dirs, err := prepareFolders(src, filepath.Join(src, "clips"))
		if err != nil || !dirs.outputInsideSource {
			t.Errorf("got %+v, %v; want outputInsideSource", dirs, err)
		}
	})
	t.Run("rejects the source as output", func(t *testing.T) {
		if _, err := prepareFolders(src, src+string(filepath.Separator)); err == nil {
			t.Error("want an error")
		}
	})
	t.Run("rejects a symlink to the source as output", func(t *testing.T) {
		link := filepath.Join(root, "link")
		if err := os.Symlink(src, link); err != nil {
			t.Skip("cannot create symlinks here:", err)
		}
		if _, err := prepareFolders(src, link); err == nil {
			t.Error("want an error")
		}
	})
	t.Run("reports a missing source", func(t *testing.T) {
		_, err := prepareFolders(filepath.Join(root, "missing"), filepath.Join(root, "out"))
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("err = %v, want 'not found'", err)
		}
	})
	t.Run("rejects a file as source", func(t *testing.T) {
		file := filepath.Join(root, "file.wav")
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := prepareFolders(file, filepath.Join(root, "out")); err == nil {
			t.Error("want an error")
		}
	})
	t.Run("requires both folders", func(t *testing.T) {
		if _, err := prepareFolders("", filepath.Join(root, "out")); err == nil {
			t.Error("empty source: want an error")
		}
		if _, err := prepareFolders(src, " "); err == nil {
			t.Error("empty output: want an error")
		}
	})
}

func TestIsInside(t *testing.T) {
	base := filepath.Join(string(filepath.Separator)+"data", "src")
	tests := []struct {
		child string
		want  bool
	}{
		{filepath.Join(base, "clips"), true},
		{filepath.Join(base, "a", "b"), true},
		{base, false},
		{filepath.Join(base, ".."), false},
		{filepath.Join(base+"2", "clips"), false},
	}
	for _, tc := range tests {
		if got := isInside(tc.child, base); got != tc.want {
			t.Errorf("isInside(%q, %q) = %v, want %v", tc.child, base, got, tc.want)
		}
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
