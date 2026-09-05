package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/collectout"
	"github.com/afterdarksys/ads-missing-utils/internal/contextsnap"
	"github.com/afterdarksys/ads-missing-utils/internal/eventwhy"
	"io"
	"os"
	"time"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("eventwhy", flag.ContinueOnError)
	input := fs.String("input", "", "Docker NDJSON file; default stdin")
	collect := fs.Bool("collect", false, "collect a bounded historical Docker event window")
	dockerContext := fs.String("context", "", "Docker context name; required for live collection")
	window := fs.Duration("window", 5*time.Minute, "historical window, maximum 24h")
	timeout := fs.Duration("timeout", 20*time.Second, "collection timeout, maximum 2m")
	maximum := fs.Int("max-events", 2000, "event limit, maximum 10000")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *version {
		fmt.Println(cli.Version)
		return 0
	}
	if fs.NArg() != 0 || *maximum < 1 || *maximum > 10000 || *window <= 0 || *window > 24*time.Hour || *timeout <= 0 || *timeout > 2*time.Minute {
		fmt.Fprintln(os.Stderr, "invalid bounds or arguments")
		return 2
	}
	var result eventwhy.Result
	if *collect {
		if *input != "" || !contextsnap.Safe(*dockerContext) {
			fmt.Fprintln(os.Stderr, "--collect requires --context and cannot use --input")
			return 2
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		result = eventwhy.Collect(ctx, *dockerContext, *window, *maximum, collectout.Run)
	} else {
		var reader io.Reader = os.Stdin
		if *input != "" {
			f, err := os.Open(*input)
			if err != nil {
				fmt.Fprintln(os.Stderr, "cannot open events")
				return 4
			}
			defer f.Close()
			reader = f
		}
		result = eventwhy.Normalize(reader, "provided", *dockerContext, *maximum)
	}
	if err := cli.WriteJSON(os.Stdout, result); err != nil {
		return 4
	}
	if result.Outcome != "pass" {
		return 3
	}
	return 0
}
