// Package birthtime resolves the creation ("birth") time of a file.
//
// The notion of a file's creation time is not exposed uniformly by the Go
// standard library across platforms, and some filesystems don't record it
// at all. Get returns the real birth time where the platform and filesystem
// support it, and falls back to the file's modification time otherwise.
package birthtime

import (
	"os"
	"time"
)

// Get returns the creation time of the file at path described by info.
func Get(path string, info os.FileInfo) time.Time {
	return get(path, info)
}
