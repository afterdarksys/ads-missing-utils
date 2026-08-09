package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/bundleinfo"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("bundleinfo", flag.ContinueOnError)
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
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: bundleinfo [--version] PATH.app|PATH.pkg")
		return cli.ExitUsage
	}
	report, err := bundleinfo.Inspect(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitRuntime
	}
	if err := cli.WriteJSON(os.Stdout, report); err != nil {
		return cli.ExitRuntime
	}
	return 0
}
