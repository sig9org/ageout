package purge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RemoveEmptyDirs removes empty directories below root. When recursive is
// false, only directories directly below root are considered. The root itself
// is never removed. In dry-run mode, directories that would become empty are
// reported without changing the filesystem. removedFiles contains files
// already removed by the purge pass, or files that would be removed when
// dryRun is true.
func RemoveEmptyDirs(w io.Writer, root string, recursive, dryRun bool, removedFiles []string, debugf func(string, ...any)) ([]string, error) {
	if debugf == nil {
		debugf = func(string, ...any) {}
	}
	removedFile := make(map[string]struct{}, len(removedFiles))
	for _, path := range removedFiles {
		removedFile[filepath.Clean(path)] = struct{}{}
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var removed []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		empty, err := removeEmptyDir(w, path, recursive, dryRun, removedFile, debugf, &removed)
		if err != nil {
			return removed, err
		}
		debugf("evaluated directory %s: empty=%v", path, empty)
	}
	return removed, nil
}

// removeEmptyDir returns whether path is empty after accounting for child
// directories removed (or that would be removed in dry-run mode).
func removeEmptyDir(w io.Writer, path string, recursive, dryRun bool, removedFiles map[string]struct{}, debugf func(string, ...any), removed *[]string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	empty := true
	for _, entry := range entries {
		child := filepath.Join(path, entry.Name())
		if _, removed := removedFiles[filepath.Clean(child)]; removed {
			continue
		}
		if !recursive || !entry.IsDir() {
			empty = false
			continue
		}

		childEmpty, err := removeEmptyDir(w, child, true, dryRun, removedFiles, debugf, removed)
		if err != nil {
			return false, err
		}
		debugf("evaluated directory %s: empty=%v", child, childEmpty)
		if !childEmpty {
			empty = false
		}
	}

	if !empty {
		return false, nil
	}
	if dryRun {
		fmt.Fprintf(w, "[dry-run] %s (empty directory)\n", path)
	} else {
		if err := os.Remove(path); err != nil {
			return false, fmt.Errorf("remove empty directory %s: %w", path, err)
		}
		fmt.Fprintf(w, "[rmdir] %s\n", path)
	}
	*removed = append(*removed, path)
	return true, nil
}
