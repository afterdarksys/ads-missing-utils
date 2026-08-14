// Package systemd collects bounded, read-only evidence about a systemd unit.
package systemd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

const Schema = "missing-utils/unitwhy/v1"

type Unit struct {
	Schema       string            `json:"schema"`
	Unit         string            `json:"unit"`
	Outcome      string            `json:"outcome"`
	Properties   map[string]string `json:"properties,omitempty"`
	FragmentPath string            `json:"fragment_path,omitempty"`
	DropIns      []string          `json:"drop_ins,omitempty"`
	Hardening    map[string]string `json:"hardening,omitempty"`
	Journal      []string          `json:"journal,omitempty"`
	Diagnostics  []cli.Diagnostic  `json:"diagnostics,omitempty"`
	Conclusion   string            `json:"conclusion"`
}

var run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func Inspect(ctx context.Context, unit string, journalLines int) (Unit, error) {
	if runtime.GOOS != "linux" {
		return Unit{}, cli.NewError(cli.ExitRuntime, "unitwhy is currently implemented for Linux systems with systemd")
	}
	if unit == "" || strings.ContainsRune(unit, '/') || strings.ContainsRune(unit, '\x00') {
		return Unit{}, cli.NewError(cli.ExitUsage, "unit must be a systemd unit name")
	}
	if journalLines < 0 || journalLines > 200 {
		return Unit{}, cli.NewError(cli.ExitUsage, "--journal-lines must be between 0 and 200")
	}
	r := Unit{Schema: Schema, Unit: unit, Outcome: "partial", Properties: map[string]string{}, Hardening: map[string]string{}}
	properties := []string{"Id", "LoadState", "ActiveState", "SubState", "Result", "MainPID", "ExecMainStatus", "ExecMainCode", "FragmentPath", "DropInPaths", "ExecStart", "EnvironmentFiles", "Restart", "RestartUSec", "MemoryCurrent", "MemoryMax", "CPUQuotaPerSecUSec", "User", "Group", "PrivateTmp", "ProtectSystem", "ProtectHome", "NoNewPrivileges", "CapabilityBoundingSet"}
	args := []string{"show", "--no-pager", "--property=" + strings.Join(properties, ","), unit}
	output, err := run(ctx, "systemctl", args...)
	if err != nil {
		return Unit{}, cli.NewError(cli.ExitRuntime, "systemctl show %s: %v", unit, err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok && value != "" {
			r.Properties[key] = value
		}
	}
	r.FragmentPath = r.Properties["FragmentPath"]
	if paths := r.Properties["DropInPaths"]; paths != "" {
		r.DropIns = strings.Fields(paths)
		sort.Strings(r.DropIns)
	}
	for _, key := range []string{"User", "Group", "PrivateTmp", "ProtectSystem", "ProtectHome", "NoNewPrivileges", "CapabilityBoundingSet", "MemoryMax", "CPUQuotaPerSecUSec"} {
		if value := r.Properties[key]; value != "" {
			r.Hardening[key] = value
		}
	}
	if journalLines > 0 {
		journal, journalErr := run(ctx, "journalctl", "--no-pager", "--output=short-iso", "--unit", unit, "--lines", fmt.Sprint(journalLines))
		if journalErr != nil {
			r.Diagnostics = append(r.Diagnostics, cli.Diagnostic{Code: "journal_unavailable", Message: journalErr.Error()})
		} else {
			r.Journal = boundedLines(journal, journalLines)
		}
	}
	state := r.Properties["ActiveState"]
	result := r.Properties["Result"]
	switch {
	case state == "active" && (result == "" || result == "success"):
		r.Outcome, r.Conclusion = "pass", "unit is active; effective service-manager evidence collected"
	case state == "failed" || result == "failed":
		r.Outcome, r.Conclusion = "fail", "unit is failed; inspect Result, ExecMainStatus, and journal evidence"
	default:
		r.Conclusion = "unit state is not fully healthy or complete evidence is unavailable"
	}
	if r.FragmentPath == "" {
		r.Diagnostics = append(r.Diagnostics, cli.Diagnostic{Code: "fragment_unavailable", Message: "systemd did not report a unit fragment path"})
	}
	return r, nil
}

func boundedLines(data []byte, maximum int) []string {
	result := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 64<<10)
	for scanner.Scan() && len(result) < maximum {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			result = append(result, line)
		}
	}
	return result
}

// UnitFiles returns candidates for a user-visible unit path and is retained as
// filesystem evidence when systemctl omits a fragment path.
func UnitFiles(unit string) []string {
	paths := []string{}
	for _, dir := range []string{"/etc/systemd/system", "/run/systemd/system", "/usr/lib/systemd/system", "/lib/systemd/system"} {
		path := filepath.Join(dir, unit)
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func NowContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
