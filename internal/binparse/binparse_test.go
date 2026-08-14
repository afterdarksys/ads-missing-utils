package binparse

import (
	"os"
	"testing"
)

func TestParseCurrentExecutable(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	report, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != Schema || report.Format == "" || len(report.Sections) == 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
}
