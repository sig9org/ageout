//go:build !darwin && !windows && !linux

package birthtime

import (
	"os"
	"time"
)

func get(_ string, info os.FileInfo) time.Time {
	return info.ModTime()
}
