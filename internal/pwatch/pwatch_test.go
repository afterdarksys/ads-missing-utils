package pwatch

import (
	"os"
	"runtime"
	"testing"
)

func TestSampleCurrentProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only")
	}
	record, err := Sample(os.Getpid())
	if err != nil || record.Status != "ok" || record.PID != os.Getpid() {
		t.Fatalf("%#v %v", record, err)
	}
}
