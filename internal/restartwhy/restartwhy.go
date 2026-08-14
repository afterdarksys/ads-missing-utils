// Package restartwhy identifies processes retaining deleted executable or
// library mappings after an update. It performs read-only /proc inspection.
package restartwhy

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

const Schema = "missing-utils/restartwhy/v1"

type Process struct {
	PID             int      `json:"pid"`
	Command         string   `json:"command,omitempty"`
	Executable      string   `json:"executable,omitempty"`
	DeletedMappings []string `json:"deleted_mappings,omitempty"`
	State           string   `json:"state"`
}
type Report struct {
	Schema      string           `json:"schema"`
	Outcome     string           `json:"outcome"`
	Processes   []Process        `json:"processes"`
	Conclusion  string           `json:"conclusion"`
	Diagnostics []cli.Diagnostic `json:"diagnostics,omitempty"`
}

func Inspect(pid int) (Report, error) {
	if runtime.GOOS != "linux" {
		return Report{}, cli.NewError(cli.ExitRuntime, "restartwhy is currently implemented for Linux")
	}
	if pid < 0 {
		return Report{}, cli.NewError(cli.ExitUsage, "--pid must be positive")
	}
	r := Report{Schema: Schema, Outcome: "pass", Processes: []Process{}}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return Report{}, cli.NewError(cli.ExitRuntime, "read /proc: %v", err)
	}
	for _, entry := range entries {
		candidate, err := strconv.Atoi(entry.Name())
		if err != nil || (pid != 0 && candidate != pid) {
			continue
		}
		p, err := inspectPID(candidate)
		if err != nil {
			r.Diagnostics = append(r.Diagnostics, cli.Diagnostic{Code: "process_unavailable", Message: err.Error(), Path: fmt.Sprintf("/proc/%d", candidate)})
			continue
		}
		if len(p.DeletedMappings) > 0 {
			r.Processes = append(r.Processes, p)
		}
	}
	if len(r.Processes) > 0 {
		r.Outcome, r.Conclusion = "fail", "processes retain deleted executable or library mappings and should be assessed for restart"
	} else {
		r.Conclusion = "no deleted executable or library mappings are visible"
	}
	if len(r.Diagnostics) > 0 && r.Outcome == "pass" {
		r.Outcome, r.Conclusion = "partial", "no deleted mappings are visible, but some processes were not inspectable"
	}
	return r, nil
}

func inspectPID(pid int) (Process, error) {
	p := Process{PID: pid, State: "current"}
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return p, err
	}
	p.Command = strings.TrimSpace(string(comm))
	if target, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
		p.Executable = target
	}
	f, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return p, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	seen := map[string]bool{}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			continue
		}
		path := strings.Join(fields[5:], " ")
		if strings.HasSuffix(path, " (deleted)") && !seen[path] {
			seen[path] = true
			p.DeletedMappings = append(p.DeletedMappings, path)
		}
	}
	if err := scanner.Err(); err != nil {
		return p, err
	}
	return p, nil
}

func IsDeletedPath(path string) bool { return strings.HasSuffix(filepath.Clean(path), " (deleted)") }
