package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/jsondiff"
	"os"
)

type stringsFlag []string

func (v *stringsFlag) String() string     { return "" }
func (v *stringsFlag) Set(s string) error { *v = append(*v, s); return nil }
func main()                               { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("jsondiff", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	desired := fs.String("desired", "", "desired JSON document")
	observed := fs.String("observed", "", "observed JSON document")
	format := fs.String("format", "json", "json or ndjson")
	tolerance := fs.Float64("numeric-tolerance", 0, "allowed numeric difference")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	var ignore, setPaths stringsFlag
	fs.Var(&ignore, "ignore", "JSON Pointer to ignore (repeatable)")
	fs.Var(&setPaths, "set-path", "JSON Pointer array treated as a set (repeatable)")
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--desired": true, "--observed": true, "--format": true, "--numeric-tolerance": true, "--ignore": true, "--set-path": true})
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return 0
	}
	if *desired == "" || *observed == "" || len(fs.Args()) != 0 || (*format != "json" && *format != "ndjson") || *tolerance < 0 {
		fmt.Fprintln(os.Stderr, "--desired and --observed are required; format must be json or ndjson; tolerance must be nonnegative")
		return cli.ExitUsage
	}
	left, err := jsondiff.DecodeFile(*desired)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	right, err := jsondiff.DecodeFile(*observed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	options := jsondiff.Options{Ignore: map[string]bool{}, SetPaths: map[string]bool{}, NumericTolerance: *tolerance}
	for _, path := range ignore {
		options.Ignore[path] = true
	}
	for _, path := range setPaths {
		options.SetPaths[path] = true
	}
	report := jsondiff.Compare(left, right, options)
	if *format == "ndjson" {
		for _, change := range report.Changes {
			if err := cli.WriteJSON(os.Stdout, change); err != nil {
				return cli.ExitRuntime
			}
		}
	} else if err := cli.WriteJSON(os.Stdout, report); err != nil {
		return cli.ExitRuntime
	}
	if report.Outcome == "pass" {
		return 0
	}
	return cli.ExitFailure
}
