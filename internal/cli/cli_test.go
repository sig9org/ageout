package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sig9org/ageout/internal/version"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// chdir switches the process working directory to dir for the duration of
// the test and restores it afterwards.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("restore Chdir: %v", err)
		}
	})
}

// fixedAge makes every file look expired relative to now.
func fixedAge(now time.Time) func(string, os.FileInfo) time.Time {
	return func(string, os.FileInfo) time.Time {
		return now.Add(-24 * time.Hour)
	}
}

func sortedLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	sort.Strings(lines)
	return lines
}

func TestRun_NoArgsShowsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(Config{Args: []string{}, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(stdout.String(), "usage: ageout") {
		t.Errorf("stdout = %q, want it to contain usage text", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRun_HelpFlag(t *testing.T) {
	for _, flagName := range []string{"-h", "--help", "-help"} {
		t.Run(flagName, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run(Config{Args: []string{flagName}, Stdout: &stdout, Stderr: &stderr})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			out := stdout.String()
			if !strings.Contains(out, "usage: ageout") {
				t.Errorf("stdout = %q, want it to contain usage text", out)
			}
			if !strings.Contains(out, "ageout "+version.Version+"\n") {
				t.Errorf("stdout = %q, want it to show the tool name and version", out)
			}
			if strings.Contains(out, "commit") {
				t.Errorf("stdout = %q, want no commit ID", out)
			}
		})
	}
}

func TestRun_HelpOtherFlagsAreAlphabetical(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(Config{Args: []string{"-help"}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	otherFlags := strings.SplitN(stdout.String(), "Other flags:\n", 2)
	if len(otherFlags) != 2 {
		t.Fatalf("stdout = %q, want Other flags section", stdout.String())
	}
	wantOrder := []string{"-created", "-debug", "-dryrun", "-help", "-recursive", "-rmdir", "-size-gte", "-size-lte", "-silent", "-update", "-version"}
	previous := -1
	for _, flagName := range wantOrder {
		index := strings.Index(otherFlags[1], flagName)
		if index < 0 {
			t.Fatalf("Other flags = %q, want %s", otherFlags[1], flagName)
		}
		if index <= previous {
			t.Fatalf("Other flags = %q, want alphabetical order %v", otherFlags[1], wantOrder)
		}
		previous = index
	}
}

func TestRun_VersionFlag(t *testing.T) {
	for _, flagName := range []string{"-v", "--version", "-version"} {
		t.Run(flagName, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run(Config{Args: []string{flagName}, Stdout: &stdout, Stderr: &stderr})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			got := strings.TrimSpace(stdout.String())
			want := "ageout " + version.Version
			if got != want {
				t.Errorf("stdout = %q, want %q", got, want)
			}
		})
	}
}

func TestRun_RequiresAgeFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(Config{Args: []string{"-r"}, Stdout: &stdout, Stderr: &stderr})
	if err == nil {
		t.Fatal("expected an error when no age flag is given")
	}
}

func TestRun_SizeFiltersCanBeUsedWithoutAge(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "small"), []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "large"), []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	var stdout, stderr bytes.Buffer
	err := Run(Config{Args: []string{"--size-gte", "5B", "--size-lte", "10"}, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := sortedLines(stdout.String()); !equalStrings(got, []string{
		"[delete] large (elapsed: 0d0h0m)",
		"[skip] small (elapsed: 0d0h0m)",
	}) {
		t.Errorf("stdout = %v, want only the file satisfying both size conditions", got)
	}
}

func TestRun_SizeAndAgeFiltersAreCombined(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old-small"), []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new-large"), []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args: []string{"--day", "1", "--size-gte", "5"}, Stdout: &stdout, Stderr: &stderr,
		Now: now,
		GetFileTime: func(path string, _ os.FileInfo) time.Time {
			if filepath.Base(path) == "old-small" {
				return now.Add(-48 * time.Hour)
			}
			return now
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := sortedLines(stdout.String()); !equalStrings(got, []string{
		"[skip] new-large (elapsed: 0d0h0m)",
		"[skip] old-small (elapsed: 2d0h0m)",
	}) {
		t.Errorf("stdout = %v, want both files skipped by one of the combined conditions", got)
	}
}

func TestRun_DefaultDirectoryIsCurrentDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	writeFile(t, filepath.Join(dir, "sub", "b.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1"},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"[delete] a.txt (elapsed: 1d0h0m)"}
	if got := sortedLines(stdout.String()); !equalStrings(got, want) {
		t.Errorf("stdout = %v, want %v", got, want)
	}
}

func TestRun_ExplicitDirectoryTarget(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "logs")
	writeFile(t, filepath.Join(sub, "a.log"))
	writeFile(t, filepath.Join(dir, "outside.log"))

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1", sub},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	want := fmt.Sprintf("[delete] %s (elapsed: 1d0h0m)", filepath.Join(sub, "a.log"))
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRun_RegexTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.log"))
	writeFile(t, filepath.Join(dir, "b.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1", `\.log$`},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	want := "[delete] a.log (elapsed: 1d0h0m)"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRun_RecursiveFlag(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	writeFile(t, filepath.Join(dir, "sub", "b.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1", "-r"},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{
		"[delete] a.txt (elapsed: 1d0h0m)",
		fmt.Sprintf("[delete] %s (elapsed: 1d0h0m)", filepath.Join("sub", "b.txt")),
	}
	sort.Strings(want)
	if got := sortedLines(stdout.String()); !equalStrings(got, want) {
		t.Errorf("stdout = %v, want %v", got, want)
	}
}

func TestRun_RmdirRemovesEmptyDirectoriesAfterFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "empty-after-purge", "old.txt"))
	writeFile(t, filepath.Join(dir, "kept", "fresh.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"--day", "1", "-recursive", "-rmdir"},
		Stdout: &stdout,
		Stderr: &stderr,
		Now:    now,
		GetFileTime: func(path string, _ os.FileInfo) time.Time {
			if filepath.Base(path) == "old.txt" {
				return now.Add(-24 * time.Hour)
			}
			return now
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "empty-after-purge")); !os.IsNotExist(statErr) {
		t.Errorf("empty-after-purge should have been removed, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "kept")); statErr != nil {
		t.Errorf("kept should remain, stat err = %v", statErr)
	}
	if !strings.Contains(stdout.String(), "[rmdir] empty-after-purge") {
		t.Errorf("stdout = %q, want removed directory status", stdout.String())
	}
}

func TestRun_RmdirWithoutRecursiveOnlyRemovesDirectEmptyDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "direct-empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "parent", "nested-empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFile(t, filepath.Join(dir, "fresh.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1", "-rmdir"},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: func(string, os.FileInfo) time.Time { return now },
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "direct-empty")); !os.IsNotExist(statErr) {
		t.Errorf("direct-empty should have been removed, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "parent", "nested-empty")); statErr != nil {
		t.Errorf("nested-empty should remain without -recursive, stat err = %v", statErr)
	}
}

func TestRun_ShortAgeFlags(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"-y", "0", "-m", "0", "-d", "1", "-H", "0", "-M", "0"},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "[delete] a.txt (elapsed: 1d0h0m)"
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRun_DryRunLeavesFilesInPlace(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	writeFile(t, f)
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1", "--dryrun"},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(stdout.String(), "a.txt") {
		t.Errorf("stdout = %q, want it to mention a.txt", stdout.String())
	}
	if _, statErr := os.Stat(f); statErr != nil {
		t.Errorf("dry-run must not delete files, stat err = %v", statErr)
	}
}

func TestRun_KeepsFreshFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"--day", "365"},
		Stdout: &stdout,
		Stderr: &stderr,
		Now:    now,
		GetFileTime: func(string, os.FileInfo) time.Time {
			return now // created "now", far newer than a 365-day-old cutoff
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "[skip] a.txt (elapsed: 0d0h0m)"
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Errorf("stdout = %q, want %q (nothing should be expired, but it's still reported by default)", got, want)
	}
}

func TestRun_SilentFlag(t *testing.T) {
	for _, flagName := range []string{"-silent", "--silent"} {
		t.Run(flagName, func(t *testing.T) {
			dir := t.TempDir()
			f := filepath.Join(dir, "a.txt")
			writeFile(t, f)
			chdir(t, dir)

			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			var stdout, stderr bytes.Buffer
			err := Run(Config{
				Args:        []string{"--day", "1", flagName},
				Stdout:      &stdout,
				Stderr:      &stderr,
				Now:         now,
				GetFileTime: fixedAge(now),
			})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if stdout.String() != "" {
				t.Errorf("stdout = %q, want empty output with %s", stdout.String(), flagName)
			}
			if _, statErr := os.Stat(f); !os.IsNotExist(statErr) {
				t.Errorf("a.txt should have been deleted even though output was suppressed, stat err = %v", statErr)
			}
		})
	}
}

// TestRun_DefaultUsesLastModifiedTime proves that, absent --created, ageout
// judges age by mtime: changing only the file's last modified time (mtime
// is unaffected by creation/birth time) is enough to make it a target.
func TestRun_DefaultUsesLastModifiedTime(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	writeFile(t, f)
	chdir(t, dir)

	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(f, old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"--day", "1", "--dryrun"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Elapsed is computed against the real clock, so only the day-scale
	// magnitude (~2 days) is asserted, not an exact minute.
	got := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(got, "[dry-run] a.txt (elapsed: 2d") {
		t.Errorf("stdout = %q, want it to start with %q (default should use last modified time)", got, "[dry-run] a.txt (elapsed: 2d")
	}
}

// TestRun_CreatedFlagParses confirms -c/--created is accepted and does not
// interfere with an explicit GetFileTime override (used elsewhere to pin
// down age without depending on real filesystem timestamps).
func TestRun_CreatedFlagParses(t *testing.T) {
	for _, flagName := range []string{"-c", "--created"} {
		t.Run(flagName, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "a.txt"))
			chdir(t, dir)

			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			var stdout, stderr bytes.Buffer
			err := Run(Config{
				Args:        []string{"--day", "1", flagName},
				Stdout:      &stdout,
				Stderr:      &stderr,
				Now:         now,
				GetFileTime: fixedAge(now),
			})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			want := "[delete] a.txt (elapsed: 1d0h0m)"
			if got := strings.TrimSpace(stdout.String()); got != want {
				t.Errorf("stdout = %q, want %q", got, want)
			}
		})
	}
}

// TestRun_CreatedFlagKeepsFreshFile is a portable smoke test for --created:
// a just-created file's creation time is always recent, so it must be kept
// regardless of platform. (Backdating mtime to test the divergent case isn't
// portable: on filesystems enforcing birth <= mtime, such as this one,
// pushing mtime into the past drags the reported creation time back with
// it, so the two criteria can't be pried apart via os.Chtimes.)
func TestRun_CreatedFlagKeepsFreshFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	writeFile(t, f)
	chdir(t, dir)

	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"--day", "1", "--created"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "[skip] a.txt (elapsed: 0d0h0m)"
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Errorf("stdout = %q, want %q (creation time is recent, so the file should be kept)", got, want)
	}
	if _, statErr := os.Stat(f); statErr != nil {
		t.Errorf("file should still exist, stat err = %v", statErr)
	}
}

func TestRun_HelpMentionsCreatedFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(Config{Args: []string{"--help"}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(stdout.String(), "-created") {
		t.Errorf("stdout = %q, want it to mention -created", stdout.String())
	}
}

func TestRun_HelpMentionsSilentFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(Config{Args: []string{"--help"}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(stdout.String(), "-silent") {
		t.Errorf("stdout = %q, want it to mention -silent", stdout.String())
	}
}

func TestRun_HelpMentionsRmdirFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(Config{Args: []string{"--help"}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(stdout.String(), "-rmdir") {
		t.Errorf("stdout = %q, want it to mention -rmdir", stdout.String())
	}
}

func TestRun_DebugFlagPrintsTimestampedTracing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1", "-debug"},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "[debug]") {
		t.Errorf("stdout = %q, want it to contain [debug] tracing with -debug", got)
	}
	if !strings.Contains(got, "[delete] a.txt") {
		t.Errorf("stdout = %q, want it to still contain the normal per-file status line", got)
	}
	// A debug line should be timestamped (YYYY-MM-DD HH:MM:SS.mmm prefix).
	timestamped := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3} \[debug\]`)
	if !timestamped.MatchString(got) {
		t.Errorf("stdout = %q, want at least one debug line prefixed with a timestamp", got)
	}
}

func TestRun_WithoutDebugFlagNoTracing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	chdir(t, dir)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:        []string{"--day", "1"},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Now:         now,
		GetFileTime: fixedAge(now),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(stdout.String(), "[debug]") {
		t.Errorf("stdout = %q, want no [debug] tracing without -debug", stdout.String())
	}
}

func TestRun_HelpMentionsDebugFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(Config{Args: []string{"--help"}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(stdout.String(), "-debug") {
		t.Errorf("stdout = %q, want it to mention -debug", stdout.String())
	}
}

func TestRun_HelpMentionsUpdateFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(Config{Args: []string{"--help"}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(stdout.String(), "-update") {
		t.Errorf("stdout = %q, want it to mention -update", stdout.String())
	}
}

func TestRun_UpdateFlag(t *testing.T) {
	for _, flagName := range []string{"-update", "--update"} {
		t.Run(flagName, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			var gotCurrent string
			err := Run(Config{
				Args:   []string{flagName},
				Stdout: &stdout,
				Stderr: &stderr,
				SelfUpdate: func(ctx context.Context, current string) (bool, string, error) {
					gotCurrent = current
					return true, "v9.9.9", nil
				},
			})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if gotCurrent != version.Version {
				t.Errorf("SelfUpdate called with current = %q, want %q", gotCurrent, version.Version)
			}
			want := fmt.Sprintf("ageout updated %s -> v9.9.9", version.Version)
			if got := strings.TrimSpace(stdout.String()); got != want {
				t.Errorf("stdout = %q, want %q", got, want)
			}
		})
	}
}

func TestRun_UpdateFlagAlreadyLatest(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"-update"},
		Stdout: &stdout,
		Stderr: &stderr,
		SelfUpdate: func(ctx context.Context, current string) (bool, string, error) {
			return false, current, nil
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := fmt.Sprintf("ageout %s is already the latest version", version.Version)
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRun_UpdateFlagPropagatesError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"-update"},
		Stdout: &stdout,
		Stderr: &stderr,
		SelfUpdate: func(ctx context.Context, current string) (bool, string, error) {
			return false, "", errors.New("boom")
		},
	})
	if err == nil {
		t.Fatal("expected an error when SelfUpdate fails")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want it to wrap the underlying error", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
