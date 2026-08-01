package scan

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
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

func setupTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	writeFile(t, filepath.Join(dir, "b.log"))
	writeFile(t, filepath.Join(dir, "sub", "c.txt"))
	writeFile(t, filepath.Join(dir, "sub", "d.log"))
	return dir
}

func relSorted(t *testing.T, root string, files []string) []string {
	t.Helper()
	rels := make([]string, len(files))
	for i, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil {
			t.Fatalf("Rel: %v", err)
		}
		rels[i] = filepath.ToSlash(rel)
	}
	sort.Strings(rels)
	return rels
}

func TestFiles_NonRecursive(t *testing.T) {
	dir := setupTree(t)

	got, err := Files(dir, Options{})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	want := []string{"a.txt", "b.log"}
	if rel := relSorted(t, dir, got); !equalStrings(rel, want) {
		t.Errorf("Files() = %v, want %v", rel, want)
	}
}

func TestFiles_Recursive(t *testing.T) {
	dir := setupTree(t)

	got, err := Files(dir, Options{Recursive: true})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	want := []string{"a.txt", "b.log", "sub/c.txt", "sub/d.log"}
	if rel := relSorted(t, dir, got); !equalStrings(rel, want) {
		t.Errorf("Files() = %v, want %v", rel, want)
	}
}

func TestFiles_Pattern(t *testing.T) {
	dir := setupTree(t)
	pattern := regexp.MustCompile(`\.log$`)

	got, err := Files(dir, Options{Recursive: true, Pattern: pattern})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	want := []string{"b.log", "sub/d.log"}
	if rel := relSorted(t, dir, got); !equalStrings(rel, want) {
		t.Errorf("Files() = %v, want %v", rel, want)
	}
}

func TestFiles_PatternNonRecursive(t *testing.T) {
	dir := setupTree(t)
	pattern := regexp.MustCompile(`\.log$`)

	got, err := Files(dir, Options{Pattern: pattern})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	want := []string{"b.log"}
	if rel := relSorted(t, dir, got); !equalStrings(rel, want) {
		t.Errorf("Files() = %v, want %v", rel, want)
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
