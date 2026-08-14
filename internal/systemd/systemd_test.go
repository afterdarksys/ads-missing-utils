package systemd

import (
	"context"
	"fmt"
	"runtime"
	"testing"
)

func TestInspectParsesMachineReadableEvidence(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd collector is Linux-only")
	}
	original := run
	defer func() { run = original }()
	run = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "systemctl" {
			return []byte("Id=demo.service\nActiveState=failed\nResult=failed\nFragmentPath=/etc/systemd/system/demo.service\nDropInPaths=/etc/systemd/system/demo.service.d/x.conf\nProtectSystem=strict\n"), nil
		}
		if name == "journalctl" {
			return []byte("one\ntwo\n"), nil
		}
		return nil, fmt.Errorf("unexpected %s", name)
	}
	r, err := Inspect(context.Background(), "demo.service", 2)
	if err != nil || r.Outcome != "fail" || len(r.Journal) != 2 || r.Hardening["ProtectSystem"] != "strict" {
		t.Fatalf("result=%#v err=%v", r, err)
	}
}
