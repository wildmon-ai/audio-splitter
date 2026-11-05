package main

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func touch(t *testing.T, root, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func relSlash(t *testing.T, root string, paths []string) []string {
	t.Helper()
	var out []string
	for _, p := range paths {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	return out
}

func TestCollectFiles(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{
		"a.wav", "B.WAV", "notes.txt",
		"._a.wav",       // macOS metadata file on memory cards
		".hidden/x.wav", // hidden folder
		"sub/c.flac", "sub/deeper/d.opus",
		"clips/old.wav", // output folder inside the source
		"$RECYCLE.BIN/e.wav", "System Volume Information/f.wav",
	} {
		touch(t, root, f)
	}
	var warnings []string
	got, err := collectFiles(root, filepath.Join(root, "clips"), func(m string) { warnings = append(warnings, m) })
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"B.WAV", "a.wav", "sub/c.flac", "sub/deeper/d.opus"}
	if rel := relSlash(t, root, got); !slices.Equal(rel, want) {
		t.Errorf("collectFiles() = %v, want %v", rel, want)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestCollectFilesSkipsUnreadableFolders(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions and a non-root user")
	}
	// Like "System Volume Information" at the root of a memory card: one
	// unreadable folder used to abort the whole scan with "0 clips".
	root := t.TempDir()
	touch(t, root, "good.wav")
	touch(t, root, "locked/inside.wav")
	locked := filepath.Join(root, "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	var warnings []string
	got, err := collectFiles(root, "", func(m string) { warnings = append(warnings, m) })
	if err != nil {
		t.Fatal(err)
	}
	if rel := relSlash(t, root, got); !slices.Equal(rel, []string{"good.wav"}) {
		t.Errorf("collectFiles() = %v, want [good.wav]", rel)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "locked") {
		t.Errorf("warnings = %v, want one about the locked folder", warnings)
	}
}

func TestCollectFilesFailsWhenTheRootIsMissing(t *testing.T) {
	if _, err := collectFiles(filepath.Join(t.TempDir(), "missing"), "", func(string) {}); err == nil {
		t.Error("want an error")
	}
}
