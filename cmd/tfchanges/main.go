package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/tfchanges"
)

func main() { os.Exit(run()) }

func run() int {
	fs := flag.NewFlagSet("tfchanges", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	format := fs.String("format", "ndjson", "json or ndjson")
	inputPath := fs.String("input", "", "Terraform/OpenTofu plan JSON file (default: stdin)")
	address := fs.String("address", "", "address regular expression filter")
	module := fs.String("module", "", "module regular expression filter")
	provider := fs.String("provider", "", "provider regular expression filter")
	resourceType := fs.String("type", "", "resource type regular expression filter")
	actions := fs.String("action", "", "comma-separated action filter")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--format": true, "--input": true, "--address": true, "--module": true, "--provider": true, "--type": true, "--action": true})
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
	summaryOnly := false
	if len(fs.Args()) == 1 && fs.Args()[0] == "summary" {
		summaryOnly = true
	} else if len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "expected optional summary subcommand")
		return cli.ExitUsage
	}
	filter, err := tfchanges.CompileFilter(*address, *module, *provider, *resourceType, *actions)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitUsage
	}
	input := io.Reader(os.Stdin)
	var file *os.File
	if *inputPath != "" {
		file, err = os.Open(*inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open input: %v\n", err)
			return cli.ExitRuntime
		}
		defer file.Close()
		input = file
	}
	plan, err := tfchanges.Decode(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	records := tfchanges.Normalize(plan, filter)
	if summaryOnly || *format == "json" {
		return writeJSON(tfchanges.Summarize(plan, records))
	}
	for _, record := range records {
		if err := cli.WriteJSON(os.Stdout, record); err != nil {
			return cli.ExitRuntime
		}
	}
	return cli.ExitOK
}

func writeJSON(value any) int {
	if err := cli.WriteJSON(os.Stdout, value); err != nil {
		return cli.ExitRuntime
	}
	return cli.ExitOK
}
