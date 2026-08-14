package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

type snapshot struct {
	Schema       string    `json:"schema"`
	Outcome      string    `json:"outcome"`
	CapturedAt   time.Time `json:"captured_at"`
	Host         string    `json:"host"`
	OS           string    `json:"os,omitempty"`
	Kernel       string    `json:"kernel,omitempty"`
	BootID       string    `json:"boot_id,omitempty"`
	UptimeSecond float64   `json:"uptime_seconds,omitempty"`
	GOOS         string    `json:"goos"`
	GOARCH       string    `json:"goarch"`
	Diagnostics  []string  `json:"diagnostics,omitempty"`
	Conclusion   string    `json:"conclusion"`
}

func main() { os.Exit(run()) }

func run() int {
	fs := flag.NewFlagSet("incidentsnap", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	format := fs.String("format", "json", "output format (json)")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(os.Args[1:]); err != nil || *format != "json" || len(fs.Args()) != 0 {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	report := snapshot{Schema: "missing-utils/incidentsnap/v1", Outcome: "partial", CapturedAt: time.Now().UTC(), Host: host, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Conclusion: "read-only host identity and boot-context snapshot collected; this is not a forensic acquisition"}
	report.OS = osRelease(&report.Diagnostics)
	report.Kernel = readTrim("/proc/sys/kernel/osrelease", &report.Diagnostics)
	report.BootID = readTrim("/proc/sys/kernel/random/boot_id", &report.Diagnostics)
	if uptime := readTrim("/proc/uptime", &report.Diagnostics); uptime != "" {
		_, _ = fmt.Sscan(uptime, &report.UptimeSecond)
	}
	if err := cli.WriteJSON(os.Stdout, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitRuntime
	}
	return cli.ExitPartial
}

func readTrim(path string, diagnostics *[]string) string {
	value, err := os.ReadFile(path)
	if err != nil {
		*diagnostics = append(*diagnostics, path+": "+err.Error())
		return ""
	}
	return strings.TrimSpace(string(value))
}

func osRelease(diagnostics *[]string) string {
	value := readTrim("/etc/os-release", diagnostics)
	for _, line := range strings.Split(value, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return ""
}
