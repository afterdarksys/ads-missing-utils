package collectout

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCollectorHelper(t *testing.T) {
	switch os.Getenv("MISSING_UTILS_COLLECTOR_TEST") {
	case "large":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", Limit+1024)))
		os.Exit(0)
	case "slow":
		time.Sleep(10 * time.Second)
		os.Exit(0)
	case "error":
		_, _ = os.Stderr.WriteString("secret-diagnostic")
		os.Exit(1)
	}
}
func TestRunBoundsAndDiscardsDiagnostics(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"large", "slow", "error"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("MISSING_UTILS_COLLECTOR_TEST", mode)
			timeout := 3 * time.Second
			if mode == "slow" {
				timeout = 50 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			started := time.Now()
			data, err := Run(ctx, exe, "-test.run=^TestCollectorHelper$")
			if err == nil || len(data) != 0 || strings.Contains(err.Error(), "secret-diagnostic") {
				t.Fatalf("%d %v", len(data), err)
			}
			if mode == "slow" && time.Since(started) > 2*time.Second {
				t.Fatal("timeout not enforced")
			}
		})
	}
}
