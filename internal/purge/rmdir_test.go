package purge

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveEmptyDirs_RecursiveRemovesBottomUpAndKeepsRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "parent", "child")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	var out bytes.Buffer
	removed, err := RemoveEmptyDirs(&out, root, true, false, nil, nil)
	if err != nil {
		t.Fatalf("RemoveEmptyDirs: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed = %v, want child and parent", removed)
	}
	if _, err := os.Stat(filepath.Join(root, "parent")); !os.IsNotExist(err) {
		t.Errorf("parent should have been removed, stat err = %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Errorf("root must remain, stat err = %v", err)
	}
}

func TestRemoveEmptyDirs_DryRunAccountsForFilesPendingDeletion(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "parent", "child", "old.txt")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var out bytes.Buffer
	removed, err := RemoveEmptyDirs(&out, root, true, true, []string{file}, nil)
	if err != nil {
		t.Fatalf("RemoveEmptyDirs: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed = %v, want child and parent to be reported", removed)
	}
	if strings.Count(out.String(), "[dry-run]") != 2 {
		t.Errorf("output = %q, want two dry-run directory entries", out.String())
	}
	if _, err := os.Stat(file); err != nil {
		t.Errorf("dry-run must keep the file, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "parent", "child")); err != nil {
		t.Errorf("dry-run must keep directories, stat err = %v", err)
	}
}
