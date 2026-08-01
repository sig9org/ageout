//go:build linux

package birthtime

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func get(path string, info os.FileInfo) time.Time {
	var stx unix.Statx_t
	err := unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_BTIME, &stx)
	if err == nil && stx.Mask&unix.STATX_BTIME != 0 {
		return time.Unix(stx.Btime.Sec, int64(stx.Btime.Nsec))
	}
	return info.ModTime()
}
