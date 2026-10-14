package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

// buildAllCommands builds every directory in cmd/ into one temporary directory.
func buildAllCommands(t *testing.T) (string, []string) {
	t.Helper()
	root := repositoryRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "cmd"))
	if err != nil {
		t.Fatalf("read cmd: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	binDir := t.TempDir()
	build := exec.Command("go", "build", "-o", binDir+string(filepath.Separator), "./cmd/...")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build commands: %v\n%s", err, output)
	}
	return binDir, names
}

func runBinary(t *testing.T, binary string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if exit, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exit.ExitCode()
	}
	if err != nil {
		t.Fatalf("run %s: %v", binary, err)
	}
	return stdout.String(), stderr.String(), 0
}

func TestEveryCommandPrintsVersion(t *testing.T) {
	binDir, names := buildAllCommands(t)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := runBinary(t, filepath.Join(binDir, name), "--version")
			if code != 0 || strings.TrimSpace(stdout) != cli.Version {
				t.Fatalf("--version code=%d stdout=%q stderr=%q, want code 0 and %q", code, stdout, stderr, cli.Version)
			}
		})
	}
}

func TestFingerprintCommandsRejectUnreadableContent(t *testing.T) {
	binDir, _ := buildAllCommands(t)
	dir := t.TempDir()
	for _, name := range []string{"driftwhy", "patchwhy"} {
		stdout, stderr, code := runBinary(t, filepath.Join(binDir, name), "--path", dir)
		if code != 4 || stdout != "" || stderr == "" {
			t.Errorf("%s on a directory: code=%d stdout=%q stderr=%q, want code 4, no report and an error", name, code, stdout, stderr)
		}
	}
}

func TestHashsumHelp(t *testing.T) {
	binDir, _ := buildAllCommands(t)
	for _, arg := range []string{"help", "-h", "--help"} {
		stdout, _, code := runBinary(t, filepath.Join(binDir, "hashsum"), arg)
		if code != 0 || !strings.Contains(stdout, "create") || !strings.Contains(stdout, "verify") {
			t.Errorf("hashsum %s: code=%d stdout=%q, want code 0 and the command list", arg, code, stdout)
		}
	}
	_, stderr, code := runBinary(t, filepath.Join(binDir, "hashsum"), "bogus")
	if code != cli.ExitUsage || !strings.Contains(stderr, "usage: hashsum") {
		t.Errorf("hashsum bogus: code=%d stderr=%q, want usage error", code, stderr)
	}
}
