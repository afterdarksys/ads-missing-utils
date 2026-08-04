package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAC10BuildSchemasAndReleaseTooling(t *testing.T) {
	root := repositoryRoot(t)
	for _, name := range []string{"response-v1.json", "jwalk-v1.json", "envsub-v1.json", "hashsum-v1.json", "meow-v1.json"} {
		data, err := os.ReadFile(filepath.Join(root, "schemas", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			t.Fatalf("schema %s is invalid JSON: %v", name, err)
		}
	}
	for _, name := range []string{"Makefile", ".goreleaser.yaml", ".github/workflows/ci.yml"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("missing required release asset %s: %v", name, err)
		}
	}
}

func TestBuildScriptDocumentsHelpAndBuildsEveryCommand(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "build.sh"))
	if err != nil {
		t.Fatalf("read build.sh: %v", err)
	}
	script := string(data)
	if !strings.Contains(script, "help     Show this help text.") {
		t.Fatal("build.sh does not document its help target")
	}
	if !strings.Contains(script, "for command_dir in cmd/*") {
		t.Fatal("build.sh does not build every cmd directory")
	}
	entries, err := os.ReadDir(filepath.Join(root, "cmd"))
	if err != nil {
		t.Fatalf("read cmd: %v", err)
	}
	if len(entries) < 24 {
		t.Fatalf("command count = %d, want >= 24", len(entries))
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(file))
}
