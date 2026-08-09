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
	fs := flag.NewFlagSet("clt", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "directory to scan")
	dest := fs.String("quarantine-dir", "", "quarantine destination")
	apply := fs.Bool("apply", false, "move candidates into quarantine")
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
	report, err := macossec.ScanCloneExecutables(*root)
	if err != nil {
		return fail(err)
	}
	moved := []string{}
	if *apply {
		if *dest == "" {
			fmt.Fprintln(os.Stderr, "--apply requires --quarantine-dir")
			return 2
		}
		moved, err = macossec.QuarantineCandidates(report, *dest)
		if err != nil {
			return fail(err)
		}
	}
	return write(map[string]any{"schema": "missing-utils/clt/v1", "report": report, "quarantined": moved})
}
func write(v any) int {
	if err := cli.WriteJSON(os.Stdout, v); err != nil {
		return 4
	}
	return 0
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return 4 }
