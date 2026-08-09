package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/pll"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("pll", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return cli.ExitUsage
	}
	if *version {
		fmt.Println(cli.Version)
		return 0
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: pll [--version] FILE...")
		return cli.ExitUsage
	}
	failed := false
	for _, path := range fs.Args() {
		report := pll.Lint(path)
		if err := cli.WriteJSON(os.Stdout, report); err != nil {
			return cli.ExitRuntime
		}
		for _, finding := range report.Findings {
			if finding.Severity == "error" {
				failed = true
			}
		}
	}
	if failed {
		return cli.ExitFailure
	}
	return 0
}
