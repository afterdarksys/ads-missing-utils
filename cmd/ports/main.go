package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/ports"
	"os"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("ports", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	port := fs.Int("port", 0, "listener port")
	transport := fs.String("transport", "", "tcp or udp")
	format := fs.String("format", "json", "json or ndjson")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--port": true, "--transport": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return 0
	}
	if (*transport != "" && *transport != "tcp" && *transport != "udp") || *port < 0 || *port > 65535 || (*format != "json" && *format != "ndjson") {
		fmt.Fprintln(os.Stderr, "invalid port, transport, or format")
		return 2
	}
	records, err := ports.List(ports.Options{Port: *port, Transport: *transport})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if *format == "ndjson" {
		for _, record := range records {
			if err := cli.WriteJSON(os.Stdout, record); err != nil {
				return 4
			}
		}
	} else if err := cli.WriteJSON(os.Stdout, records); err != nil {
		return 4
	}
	return 0
}
