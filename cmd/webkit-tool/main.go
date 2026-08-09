package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/macossec"
	"os"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("webkit-tool", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("scan", ".", "source tree to inspect")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *version {
		fmt.Println(cli.Version)
		return 0
	}
	report, err := macossec.ScanWebKit(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	if err := cli.WriteJSON(os.Stdout, report); err != nil {
		return 4
	}
	return 0
}
