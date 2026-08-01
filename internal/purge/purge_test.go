package purge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// fakeFileTime returns a GetFileTime stub driven by an explicit path->time
// map, decoupling tests from the real (platform-dependent) birth time.
func fakeFileTime(times map[string]time.Time) func(string, os.FileInfo) time.Time {
	return func(path string, _ os.FileInfo) time.Time {
		return times[path]
	}
}

func TestRun_DeletesExpiredFiles(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "old.txt")
	newFile := filepath.Join(dir, "new.txt")
	writeFile(t, oldFile)
	writeFile(t, newFile)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := now
	times := map[string]time.Time{
		oldFile: cutoff.Add(-time.Hour), // expired
		newFile: cutoff.Add(time.Hour),  // fresh
	}

	p := &Purger{GetFileTime: fakeFileTime(times)}
	var buf bytes.Buffer
	results, err := p.Run(&buf, []string{oldFile, newFile}, now, cutoff, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 || results[0].Path != oldFile || !results[0].Deleted {
		t.Fatalf("results = %+v, want single deleted old.txt", results)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("old.txt should have been removed, stat err = %v", err)
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Errorf("new.txt should still exist, stat err = %v", err)
	}
	want := fmt.Sprintf("[delete] %s (elapsed: 0d1h0m)\n[skip] %s (elapsed: 0d0h0m)", oldFile, newFile)
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestRun_DryRunDoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "old.txt")
	writeFile(t, oldFile)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := now
	times := map[string]time.Time{oldFile: cutoff.Add(-time.Hour)}

	p := &Purger{GetFileTime: fakeFileTime(times)}
	var buf bytes.Buffer
	results, err := p.Run(&buf, []string{oldFile}, now, cutoff, true)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 || results[0].Deleted {
		t.Fatalf("results = %+v, want single non-deleted entry", results)
	}
	if _, err := os.Stat(oldFile); err != nil {
		t.Errorf("dry-run must not delete the file, stat err = %v", err)
	}
	want := fmt.Sprintf("[dry-run] %s (elapsed: 0d1h0m)", oldFile)
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestRun_KeepsFreshFiles(t *testing.T) {
	dir := t.TempDir()
	newFile := filepath.Join(dir, "new.txt")
	writeFile(t, newFile)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := now
	times := map[string]time.Time{newFile: cutoff.Add(time.Minute)}

	p := &Purger{GetFileTime: fakeFileTime(times)}
	var buf bytes.Buffer
	results, err := p.Run(&buf, []string{newFile}, now, cutoff, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %+v, want none", results)
	}
	want := fmt.Sprintf("[skip] %s (elapsed: 0d0h0m)", newFile)
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("output = %q, want %q (fresh files are still reported when written to w)", got, want)
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Errorf("new.txt should still exist, stat err = %v", err)
	}
}

func TestRun_ExactlyAtCutoffIsExpired(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "edge.txt")
	writeFile(t, f)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := now
	times := map[string]time.Time{f: cutoff}

	p := &Purger{GetFileTime: fakeFileTime(times)}
	var buf bytes.Buffer
	results, err := p.Run(&buf, []string{f}, now, cutoff, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 || !results[0].Deleted {
		t.Fatalf("results = %+v, want file exactly at cutoff to be deleted", results)
	}
}

func TestRun_DebugfReceivesPerFileTracing(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "old.txt")
	writeFile(t, oldFile)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := now
	times := map[string]time.Time{oldFile: cutoff.Add(-time.Hour)}

	var traced []string
	p := &Purger{
		GetFileTime: fakeFileTime(times),
		Debugf: func(format string, args ...any) {
			traced = append(traced, fmt.Sprintf(format, args...))
		},
	}
	var buf bytes.Buffer
	if _, err := p.Run(&buf, []string{oldFile}, now, cutoff, false); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(traced) != 1 || !strings.Contains(traced[0], oldFile) {
		t.Errorf("traced = %v, want one entry mentioning %s", traced, oldFile)
	}
}

func TestRun_NilDebugfIsSafe(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	writeFile(t, f)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := now
	times := map[string]time.Time{f: cutoff.Add(-time.Hour)}

	p := &Purger{GetFileTime: fakeFileTime(times)}
	var buf bytes.Buffer
	if _, err := p.Run(&buf, []string{f}, now, cutoff, false); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestNew_DefaultsToLastModifiedTime(t *testing.T) {
	p := New(false)
	if p.GetFileTime == nil {
		t.Fatal("New(false) should set a default GetFileTime")
	}
}

func TestNew_UseCreatedUsesBirthtime(t *testing.T) {
	p := New(true)
	if p.GetFileTime == nil {
		t.Fatal("New(true) should set a default GetFileTime")
	}
}

func TestModTime_MatchesFileInfoModTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	writeFile(t, path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if got, want := ModTime(path, info), info.ModTime(); !got.Equal(want) {
		t.Errorf("ModTime() = %v, want %v", got, want)
	}
}

func TestRun_DefaultCriterionIsLastModifiedTime(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	writeFile(t, f)

	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(f, old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	p := &Purger{}
	var buf bytes.Buffer
	results, err := p.Run(&buf, []string{f}, now, cutoff, true)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v, want the file (aged via last modified time) to be a target", results)
	}
}

func TestFormatElapsed(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		t    time.Time
		want string
	}{
		{"just now", now, "0d0h0m"},
		{"minutes ago", now.Add(-30 * time.Minute), "0d0h30m"},
		{"hours ago", now.Add(-5 * time.Hour), "0d5h0m"},
		{"days, hours, and minutes ago", now.Add(-(2*24 + 3) * time.Hour).Add(-15 * time.Minute), "2d3h15m"},
		{"in the future is clamped to zero", now.Add(time.Hour), "0d0h0m"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatElapsed(now, c.t); got != c.want {
				t.Errorf("formatElapsed() = %q, want %q", got, c.want)
			}
		})
	}
}
