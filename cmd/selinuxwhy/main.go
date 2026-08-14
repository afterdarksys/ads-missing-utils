package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/selinuxwhy"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("selinuxwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	auditLog := fs.String("audit-log", "/var/log/audit/audit.log", "audit log path")
	maximum := fs.Int("max-denials", 50, "maximum AVC denials (1-1000)")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--audit-log": true, "--max-denials": true, "--format": true})); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--format json is required")
		return cli.ExitUsage
	}
	report, err := selinuxwhy.Inspect(*auditLog, *maximum)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if err := cli.WriteJSON(os.Stdout, report); err != nil {
		return cli.ExitRuntime
	}
	if report.Outcome == "fail" {
		return cli.ExitFailure
	}
	return cli.ExitPartial
}
