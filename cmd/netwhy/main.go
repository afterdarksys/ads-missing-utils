package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"net"
	"os"
	"sort"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("netwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	host := fs.String("host", "", "hostname or IP")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return 0
	}
	if *host == "" || *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--host and --format json are required")
		return 2
	}
	ips, err := net.LookupIP(*host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	values := make([]string, 0, len(ips))
	for _, ip := range ips {
		values = append(values, ip.String())
	}
	sort.Strings(values)
	cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/netwhy/v1", "outcome": "partial", "host": *host, "addresses": values, "conclusion": "DNS resolution evidence collected; route, proxy, namespace, and firewall evidence is unavailable"})
	return 3
}
