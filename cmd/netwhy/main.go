package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/netwhy"
)

func main() { os.Exit(run()) }

func run() int {
	fs := flag.NewFlagSet("netwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	host := fs.String("host", "", "hostname or IP address")
	port := fs.Int("port", 0, "optional TCP/UDP port for report context")
	format := fs.String("format", "json", "output format (json)")
	version := fs.Bool("version", false, "print version")
	_ = fs.Bool("no-color", false, "disable color")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if *host == "" || *port < 0 || *port > 65535 || *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--host is required; --port must be 0 through 65535; --format must be json")
		return cli.ExitUsage
	}
	report, err := netwhy.Inspect(context.Background(), *host, *port)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitFailure
	}
	if err := cli.WriteJSON(os.Stdout, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitRuntime
	}
	return cli.ExitPartial
}
