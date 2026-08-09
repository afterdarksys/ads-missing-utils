package pll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintLaunchdLogic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.plist")
	body := `<?xml version="1.0"?><plist version="1.0"><dict><key>Label</key><string>x</string><key>Program</key><string>/definitely/missing</string><key>RunAtLoad</key><true/><key>KeepAlive</key><true/></dict></plist>`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Lint(path)
	codes := map[string]bool{}
	for _, finding := range report.Findings {
		codes[finding.Code] = true
	}
	if !codes["PLL_PROGRAM_MISSING"] || !codes["PLL_PERSISTENT_JOB"] {
		t.Fatalf("findings %#v", report.Findings)
	}
}

func TestLintDuplicateAndMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.plist")
	if err := os.WriteFile(path, []byte(`<plist><dict><key>A</key><string>x</string><key>A</key></dict></plist>`), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Lint(path)
	if len(report.Findings) == 0 {
		t.Fatal("expected findings")
	}
}
