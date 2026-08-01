//go:build windows

package birthtime

import (
	"os"
	"syscall"
	"time"
)

func get(_ string, info os.FileInfo) time.Time {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds())
	}
	return info.ModTime()
}
