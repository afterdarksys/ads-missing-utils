package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/ports"
	"os"
)

type result struct {
	Schema     string         `json:"schema"`
	Port       int            `json:"port"`
	Outcome    string         `json:"outcome"`
	Listeners  []ports.Record `json:"listeners"`
	Conclusion string         `json:"conclusion"`
}

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("portwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	port := fs.Int("port", 0, "port to explain")
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
	if *port < 1 || *port > 65535 || *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--port (1-65535) and --format json are required")
		return 2
	}
	listeners, err := ports.List(ports.Options{Port: *port})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	outcome, conclusion := "pass", "no listener is visible on this host"
	if len(listeners) > 0 {
		conclusion = "listener ownership evidence collected"
		for _, listener := range listeners {
			if listener.Visibility == "partial" {
				outcome = "partial"
				conclusion = "a listener is visible but its owning process is not accessible"
				break
			}
		}
	}
	if err := cli.WriteJSON(os.Stdout, result{"missing-utils/portwhy/v1", *port, outcome, listeners, conclusion}); err != nil {
		return 4
	}
	if outcome == "partial" {
		return 3
	}
	return 0
}
