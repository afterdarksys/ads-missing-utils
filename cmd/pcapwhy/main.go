package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/pcapwhy"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("pcapwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	maxPackets := fs.Int("max-packets", 100000, "maximum packets per capture (1-1000000)")
	maxFlows := fs.Int("max-flows", 10000, "maximum flows per capture (1-100000)")
	format := fs.String("format", "json", "output format (json)")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if *format != "json" || len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pcapwhy [--max-packets N] [--max-flows N] [--format json] CAPTURE [CAPTURE...]")
		return cli.ExitUsage
	}
	report, err := pcapwhy.Analyze(fs.Args(), pcapwhy.Options{MaxPackets: *maxPackets, MaxFlows: *maxFlows})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if err := cli.WriteJSON(os.Stdout, report); err != nil {
		return cli.ExitRuntime
	}
	if report.Outcome == "partial" {
		return cli.ExitPartial
	}
	return cli.ExitOK
}
