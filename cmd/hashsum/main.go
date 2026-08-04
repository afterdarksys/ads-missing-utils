package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/hashsum"
)

func main() { os.Exit(run()) }
func run() int {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: hashsum create|verify [options]")
		return cli.ExitUsage
	}
	switch os.Args[1] {
	case "create":
		return create(os.Args[2:])
	case "verify":
		return verify(os.Args[2:])
	case "--version", "version":
		fmt.Println(cli.Version)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown subcommand", os.Args[1])
		return cli.ExitUsage
	}
}
func create(args []string) int {
	fs := flag.NewFlagSet("hashsum create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", "", "manifest root directory")
	output := fs.String("output", "", "manifest output file; stdout when omitted")
	fromJwalk := fs.Bool("from-jwalk", false, "read jwalk NDJSON from stdin")
	workers := fs.Int("workers", 1, "parallel file workers")
	progress := fs.Bool("progress", false, "write progress to stderr")
	format := fs.String("format", "json", "json")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args = cli.ReorderInterspersed(args, map[string]bool{"--root": true, "--output": true, "--workers": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	if *format != "json" && *format != "ndjson" {
		fmt.Fprintln(os.Stderr, "--format must be json or ndjson")
		return cli.ExitUsage
	}
	var progressWriter *os.File
	if *progress {
		progressWriter = os.Stderr
	}
	manifest, err := hashsum.Create(hashsum.CreateOptions{Root: *root, Paths: fs.Args(), FromJwalk: *fromJwalk, Workers: *workers, Progress: progressWriter}, os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if *output != "" {
		if err := hashsum.WriteManifest(*output, manifest); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return cli.ExitRuntime
		}
	} else if err := cli.WriteJSON(os.Stdout, manifest); err != nil {
		return cli.ExitRuntime
	}
	if len(manifest.Errors) > 0 {
		return cli.ExitFailure
	}
	return cli.ExitOK
}
func verify(args []string) int {
	fs := flag.NewFlagSet("hashsum verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", "", "verification root directory")
	unexpected := fs.Bool("unexpected", true, "report unexpected files")
	format := fs.String("format", "json", "json or ndjson")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args = cli.ReorderInterspersed(args, map[string]bool{"--root": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	if len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "verify requires one manifest path")
		return cli.ExitUsage
	}
	if *format != "json" && *format != "ndjson" {
		fmt.Fprintln(os.Stderr, "--format must be json or ndjson")
		return cli.ExitUsage
	}
	records, err := hashsum.Verify(hashsum.VerifyOptions{Root: *root, ManifestPath: fs.Args()[0], IncludeUnexpected: *unexpected})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if *format == "ndjson" {
		for _, record := range records {
			if err := cli.WriteJSON(os.Stdout, record); err != nil {
				return cli.ExitRuntime
			}
		}
	} else if err := cli.WriteJSON(os.Stdout, records); err != nil {
		return cli.ExitRuntime
	}
	for _, record := range records {
		if record.Status != "ok" {
			return cli.ExitFailure
		}
	}
	return cli.ExitOK
}
