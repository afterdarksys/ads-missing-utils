package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/systemd"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("unitwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	unit := fs.String("unit", "", "systemd unit name")
	lines := fs.Int("journal-lines", 20, "recent journal lines (0-200)")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--unit": true, "--journal-lines": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if *unit == "" || *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--unit and --format json are required")
		return cli.ExitUsage
	}
	ctx, cancel := systemd.NowContext()
	defer cancel()
	report, err := systemd.Inspect(ctx, *unit, *lines)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if err := cli.WriteJSON(os.Stdout, report); err != nil {
		return cli.ExitRuntime
	}
	switch report.Outcome {
	case "pass":
		return cli.ExitOK
	case "fail":
		return cli.ExitFailure
	default:
		return cli.ExitPartial
	}
}
