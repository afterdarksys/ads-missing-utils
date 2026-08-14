// Package selinuxwhy reads AVC denial records as evidence without modifying
// policy, labels, booleans, or enforcement state.
package selinuxwhy

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

const Schema = "missing-utils/selinuxwhy/v1"

type Denial struct {
	Raw         string   `json:"raw"`
	PID         string   `json:"pid,omitempty"`
	Comm        string   `json:"comm,omitempty"`
	Name        string   `json:"name,omitempty"`
	SContext    string   `json:"scontext,omitempty"`
	TContext    string   `json:"tcontext,omitempty"`
	TClass      string   `json:"tclass,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}
type Report struct {
	Schema      string           `json:"schema"`
	Outcome     string           `json:"outcome"`
	Enforcing   string           `json:"enforcing"`
	AuditLog    string           `json:"audit_log"`
	Denials     []Denial         `json:"denials"`
	Diagnostics []cli.Diagnostic `json:"diagnostics,omitempty"`
	Conclusion  string           `json:"conclusion"`
}

var field = regexp.MustCompile(`\b(pid|comm|name|scontext|tcontext|tclass)=([^\s]+)`)
var denied = regexp.MustCompile(`avc:\s+denied\s+\{([^}]+)\}`)

func Inspect(logPath string, maximum int) (Report, error) {
	if runtime.GOOS != "linux" {
		return Report{}, cli.NewError(cli.ExitRuntime, "selinuxwhy is currently implemented for Linux")
	}
	if maximum < 1 || maximum > 1000 {
		return Report{}, cli.NewError(cli.ExitUsage, "--max-denials must be between 1 and 1000")
	}
	if logPath == "" {
		logPath = "/var/log/audit/audit.log"
	}
	r := Report{Schema: Schema, Outcome: "partial", AuditLog: logPath, Denials: []Denial{}}
	if data, err := os.ReadFile("/sys/fs/selinux/enforce"); err == nil {
		if strings.TrimSpace(string(data)) == "1" {
			r.Enforcing = "enforcing"
		} else {
			r.Enforcing = "permissive"
		}
	} else {
		r.Enforcing = "unknown"
		r.Diagnostics = append(r.Diagnostics, cli.Diagnostic{Code: "selinux_status_unavailable", Message: err.Error(), Path: "/sys/fs/selinux/enforce"})
	}
	f, err := os.Open(logPath)
	if err != nil {
		r.Diagnostics = append(r.Diagnostics, cli.Diagnostic{Code: "audit_log_unavailable", Message: err.Error(), Path: logPath})
		r.Conclusion = "SELinux audit evidence is unavailable; inspect audit-log permissions and SELinux status"
		return r, nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "avc:  denied") {
			continue
		}
		r.Denials = append(r.Denials, parse(line))
		if len(r.Denials) >= maximum {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return Report{}, fmt.Errorf("read audit log: %w", err)
	}
	if len(r.Denials) > 0 {
		r.Outcome, r.Conclusion = "fail", "AVC denials were found; inspect source/target contexts and permissions before changing policy"
	} else if r.Enforcing == "unknown" {
		r.Conclusion = "no readable AVC denials found; SELinux status is unavailable"
	} else {
		r.Conclusion = "no AVC denials found in the inspected audit log"
	}
	return r, nil
}

func parse(line string) Denial {
	d := Denial{Raw: line}
	if match := denied.FindStringSubmatch(line); len(match) == 2 {
		d.Permissions = strings.Fields(match[1])
	}
	for _, match := range field.FindAllStringSubmatch(line, -1) {
		switch match[1] {
		case "pid":
			d.PID = match[2]
		case "comm":
			d.Comm = strings.Trim(match[2], `"`)
		case "name":
			d.Name = strings.Trim(match[2], `"`)
		case "scontext":
			d.SContext = match[2]
		case "tcontext":
			d.TContext = match[2]
		case "tclass":
			d.TClass = match[2]
		}
	}
	return d
}
