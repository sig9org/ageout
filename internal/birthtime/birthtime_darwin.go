//go:build darwin

package birthtime

import (
	"os"
	"syscall"
	"time"
)

func get(_ string, info os.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Birthtimespec.Sec, st.Birthtimespec.Nsec)
	}
	return info.ModTime()
}
