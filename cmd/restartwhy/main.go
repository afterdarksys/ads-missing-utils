package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/restartwhy"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("restartwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pid := fs.Int("pid", 0, "inspect one process ID (default: all visible processes)")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--pid": true, "--format": true})); err != nil {
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
	report, err := restartwhy.Inspect(*pid)
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
	if report.Outcome == "partial" {
		return cli.ExitPartial
	}
	return cli.ExitOK
}
