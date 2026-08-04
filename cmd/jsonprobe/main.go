package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/jsonprobe"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("jsonprobe", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	inputPath := fs.String("input", "", "JSON check specification (default: stdin)")
	format := fs.String("format", "json", "json or ndjson")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--input": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if *format != "json" && *format != "ndjson" {
		fmt.Fprintln(os.Stderr, "--format must be json or ndjson")
		return cli.ExitUsage
	}
	if len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "jsonprobe accepts no positional arguments")
		return cli.ExitUsage
	}
	input := io.Reader(os.Stdin)
	var file *os.File
	var err error
	if *inputPath != "" {
		file, err = os.Open(*inputPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return cli.ExitRuntime
		}
		defer file.Close()
		input = file
	}
	spec, err := jsonprobe.Decode(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	result := jsonprobe.Run(context.Background(), spec)
	if *format == "ndjson" {
		for _, record := range result.Checks {
			if err := cli.WriteJSON(os.Stdout, record); err != nil {
				return cli.ExitRuntime
			}
		}
	} else if err := cli.WriteJSON(os.Stdout, result); err != nil {
		return cli.ExitRuntime
	}
	if result.Outcome == "pass" {
		return cli.ExitOK
	}
	return cli.ExitFailure
}
