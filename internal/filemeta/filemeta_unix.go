//go:build !windows

package filemeta

import (
	"os"
	"syscall"
)

func valuesFromInfo(info os.FileInfo) Values {
	v := Values{Mode: info.Mode().String()}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid, gid := uint32(stat.Uid), uint32(stat.Gid)
		inode, device := uint64(stat.Ino), uint64(stat.Dev)
		v.UID, v.GID, v.Inode, v.Device = &uid, &gid, &inode, &device
	}
	return v
}
