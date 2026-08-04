package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/ports"
	"os"
)

func main() {
	fs := flag.NewFlagSet("expose", flag.ExitOnError)
	format := fs.String("format", "json", "json")
	if fs.Parse(os.Args[1:]) != nil || *format != "json" {
		os.Exit(2)
	}
	records, e := ports.List(ports.Options{})
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(cli.ExitCode(e))
	}
	cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/expose/v1", "outcome": "partial", "listeners": records, "conclusion": "local listener exposure collected; firewall and remote reachability are not yet evaluated"})
	os.Exit(3)
}
