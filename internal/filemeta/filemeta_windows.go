//go:build windows

package filemeta

import "os"

func valuesFromInfo(info os.FileInfo) Values {
	return Values{Mode: info.Mode().String()}
}
