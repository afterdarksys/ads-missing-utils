package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/pwatch"
	"os"
)

func main() {
	fs := flag.NewFlagSet("servicewhy", flag.ExitOnError)
	pid := fs.Int("pid", 0, "process ID")
	format := fs.String("format", "json", "json")
	if fs.Parse(os.Args[1:]) != nil || *pid < 1 || *format != "json" {
		os.Exit(2)
	}
	r, e := pwatch.Sample(*pid)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(cli.ExitCode(e))
	}
	cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/servicewhy/v1", "outcome": "partial", "process": r, "conclusion": "process health evidence collected; service-manager dependency and restart evidence is unavailable"})
	os.Exit(3)
}
