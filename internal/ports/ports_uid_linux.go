//go:build linux

package ports

import (
	"io/fs"
	"syscall"
)

func ownerUID(info fs.FileInfo) int {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid)
	}
	return 0
}
