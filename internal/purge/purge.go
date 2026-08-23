// Package purge decides which files have aged past a cutoff and removes
// them (or reports what would be removed in dry-run mode).
package purge

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sig9org/ageout/internal/birthtime"
)

// Result describes the outcome for a single file.
type Result struct {
	Path    string
	Deleted bool
}

// Purger evaluates and removes expired files. GetFileTime is called to
// determine each file's age; it defaults to ModTime (the file's last
// modified time) but can be overridden (e.g. in tests, or to use
// birthtime.Get) to avoid depending on real filesystem timestamps or to
// judge age by creation time instead.
type Purger struct {
	GetFileTime func(path string, info os.FileInfo) time.Time
	// IgnoreAge disables the age condition, allowing size-only filtering.
	IgnoreAge bool
	SizeGTE   *int64
	SizeLTE   *int64

	// Debugf, when set, receives per-file tracing (the file evaluated, its
	// resolved age, and the resulting decision). It is nil by default;
	// production callers wire it to logx.Logger.Debugf under -debug.
	Debugf func(format string, args ...any)
}

// New returns a Purger that resolves file age via the last modified time,
// or the creation ("birth") time when useCreated is true.
func New(useCreated bool) *Purger {
	if useCreated {
		return &Purger{GetFileTime: birthtime.Get}
	}
	return &Purger{GetFileTime: ModTime}
}

// ModTime returns the file's last modified time. This is the default
// criterion used to decide whether a file has aged past the cutoff.
func ModTime(_ string, info os.FileInfo) time.Time {
	return info.ModTime()
}

// Run evaluates each file in files against cutoff (itself computed relative
// to now). Files whose age is at or past cutoff are targets: in dry-run mode
// they are merely reported, and otherwise removed. Every file — target or
// not — is written to w with its status ("[delete]", "[dry-run]", or
// "[skip]") and its age relative to now, one line each.
func (p *Purger) Run(w io.Writer, files []string, now, cutoff time.Time, dryRun bool) ([]Result, error) {
	getFileTime := p.GetFileTime
	if getFileTime == nil {
		getFileTime = ModTime
	}
	debugf := p.Debugf
	if debugf == nil {
		debugf = func(string, ...any) {}
	}

	var results []Result
	for _, f := range files {
		info, err := os.Lstat(f)
		if err != nil {
			return results, err
		}
		if !info.Mode().IsRegular() {
			debugf("skipping non-regular file %s", f)
			continue
		}

		ft := getFileTime(f, info)
		elapsed := formatElapsed(now, ft)
		debugf("evaluating %s: fileTime=%s cutoff=%s elapsed=%s", f, ft.Format(time.RFC3339), cutoff.Format(time.RFC3339), elapsed)

		ageMatch := p.IgnoreAge || !ft.After(cutoff)
		sizeMatch := (p.SizeGTE == nil || info.Size() >= *p.SizeGTE) &&
			(p.SizeLTE == nil || info.Size() <= *p.SizeLTE)
		if !ageMatch || !sizeMatch {
			fmt.Fprintf(w, "[skip] %s (elapsed: %s)\n", f, elapsed)
			continue
		}

		if dryRun {
			fmt.Fprintf(w, "[dry-run] %s (elapsed: %s)\n", f, elapsed)
			results = append(results, Result{Path: f, Deleted: false})
			continue
		}

		if err := os.Remove(f); err != nil {
			return results, fmt.Errorf("remove %s: %w", f, err)
		}
		fmt.Fprintf(w, "[delete] %s (elapsed: %s)\n", f, elapsed)
		results = append(results, Result{Path: f, Deleted: true})
	}
	return results, nil
}

// formatElapsed renders how long ago t was, relative to now, as a compact
// "<days>d<hours>h<minutes>m" string. Negative durations (t after now) are
// clamped to zero.
func formatElapsed(now, t time.Time) string {
	d := now.Sub(t).Round(time.Minute)
	if d < 0 {
		d = 0
	}

	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute

	return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
}
