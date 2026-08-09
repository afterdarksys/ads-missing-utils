package metascore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIndexAndScoreMetadataChanges(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(t.TempDir(), "meta.json")
	path := filepath.Join(root, "file")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Index(root, state); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Check(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Anomalies) != 1 || report.Anomalies[0].Score == 0 || len(report.Anomalies[0].Evidence) < 2 {
		t.Fatalf("report %#v", report)
	}
}
