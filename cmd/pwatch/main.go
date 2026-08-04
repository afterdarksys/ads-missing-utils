package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/pwatch"
	"os"
	"time"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("pwatch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pid := fs.Int("pid", 0, "process ID")
	count := fs.Int("count", 1, "number of samples (1-3600)")
	interval := fs.Duration("interval", time.Second, "sample interval")
	format := fs.String("format", "ndjson", "json or ndjson")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--pid": true, "--count": true, "--interval": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return 0
	}
	if *pid < 1 || *count < 1 || *count > 3600 || *interval < 0 || (*format != "json" && *format != "ndjson") || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "invalid pid, count, interval, or format")
		return 2
	}
	records := []pwatch.Record{}
	for i := 0; i < *count; i++ {
		if i > 0 {
			time.Sleep(*interval)
		}
		record, err := pwatch.Sample(*pid)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return cli.ExitCode(err)
		}
		records = append(records, record)
		if *format == "ndjson" {
			if err := cli.WriteJSON(os.Stdout, record); err != nil {
				return 4
			}
		}
		if record.Status == "exited" {
			break
		}
	}
	if *format == "json" {
		if err := cli.WriteJSON(os.Stdout, records); err != nil {
			return 4
		}
	}
	return 0
}
