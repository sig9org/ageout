// Package scan locates candidate files under a root directory.
package scan

import (
	"io/fs"
	"path/filepath"
	"regexp"
)

// Options controls how Files walks a directory tree.
type Options struct {
	// Recursive, when true, descends into subdirectories. When false, only
	// files directly inside root are considered.
	Recursive bool

	// Pattern, when non-nil, restricts results to files whose base name
	// matches the regular expression. A nil Pattern matches every file.
	Pattern *regexp.Regexp
}

// Files returns the paths of regular files under root that match opts.
// Directories themselves are never included in the result.
func Files(root string, opts Options) ([]string, error) {
	var results []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && !opts.Recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if opts.Pattern != nil && !opts.Pattern.MatchString(d.Name()) {
			return nil
		}
		results = append(results, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}
