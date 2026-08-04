package planned

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run("example", "Example summary.", []string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(--help) exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "Status: planned") {
		t.Fatalf("help output does not identify planned status: %q", stderr.String())
	}
}
