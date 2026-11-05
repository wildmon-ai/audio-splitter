package main

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var audioExtensions = map[string]bool{
	".wav": true, ".mp3": true, ".flac": true,
	".ogg": true, ".m4a": true, ".opus": true,
}

// System folders found at the root of memory cards and USB drives.
var systemFolders = map[string]bool{
	"$recycle.bin":              true,
	"system volume information": true,
}

// collectFiles lists the audio files under root in a stable order.
//
// Hidden files and folders are ignored, including the "._name.wav" metadata
// files macOS writes on memory cards, as are skipDir and system folders. A
// subfolder that cannot be read is reported through warn and skipped instead
// of aborting the whole scan.
func collectFiles(root, skipDir string, warn func(string)) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			warn(fmt.Sprintf("Skipped %s: %v", relPath(root, path), shortError(err)))
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") || systemFolders[strings.ToLower(name)] ||
				(skipDir != "" && samePath(path, skipDir)) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasPrefix(name, ".") && audioExtensions[strings.ToLower(filepath.Ext(name))] {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// shortError drops the path that fs errors repeat ("open /a/b: permission denied").
func shortError(err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}
