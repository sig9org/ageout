package birthtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGet_FreshFileIsRecent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.txt")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	got := Get(path, info)
	if diff := time.Since(got); diff < 0 || diff > time.Minute {
		t.Errorf("Get() = %v, want a time within the last minute (diff %v)", got, diff)
	}
}
