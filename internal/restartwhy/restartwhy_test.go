package restartwhy

import "testing"

func TestIsDeletedPath(t *testing.T) {
	if !IsDeletedPath("/usr/lib/libexample.so (deleted)") || IsDeletedPath("/usr/lib/libexample.so") {
		t.Fatal("deleted mapping recognition failed")
	}
}
