package filemeta

import "os"

type Values struct {
	Mode   string  `json:"mode"`
	UID    *uint32 `json:"uid"`
	GID    *uint32 `json:"gid"`
	Inode  *uint64 `json:"inode"`
	Device *uint64 `json:"device"`
}

func FromInfo(info os.FileInfo) Values {
	return valuesFromInfo(info)
}
