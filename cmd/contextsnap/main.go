package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/collectout"
	"github.com/afterdarksys/ads-missing-utils/internal/contextsnap"
	"io"
	"os"
	"time"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("contextsnap", flag.ContinueOnError)
	cloud := fs.String("cloud", "", "aws or alicloud")
	profile := fs.String("profile", "", "explicit CLI profile")
	region := fs.String("region", "", "region selected for this collector; not inferred from STS")
	collect := fs.Bool("collect", false, "invoke read-only caller identity CLI")
	input := fs.String("input", "", "saved STS JSON; default stdin")
	observed := fs.String("observed-at", "", "RFC3339 observation time for saved input")
	timeout := fs.Duration("timeout", 20*time.Second, "live collection deadline; maximum 2m")
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
	if fs.NArg() != 0 || *timeout <= 0 || *timeout > 2*time.Minute {
		fmt.Fprintln(os.Stderr, "invalid arguments or timeout")
		return 2
	}
	if err := contextsnap.Validate(*cloud, *profile, *region); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	var result contextsnap.Result
	if *collect {
		if *input != "" || *observed != "" {
			fmt.Fprintln(os.Stderr, "--collect cannot be combined with saved input options")
			return 2
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		result = contextsnap.Collect(ctx, *cloud, *profile, *region, collectout.Run)
	} else {
		at, err := time.Parse(time.RFC3339Nano, *observed)
		if err != nil {
			fmt.Fprintln(os.Stderr, "saved input requires --observed-at RFC3339")
			return 2
		}
		var reader io.Reader = os.Stdin
		if *input != "" {
			f, err := os.Open(*input)
			if err != nil {
				fmt.Fprintln(os.Stderr, "cannot open identity input")
				return 4
			}
			defer f.Close()
			reader = f
		}
		data, err := io.ReadAll(io.LimitReader(reader, collectout.Limit+1))
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot read identity input")
			return 4
		}
		result = contextsnap.Normalize(data, *cloud, *profile, *region, "provided", at)
	}
	if err := cli.WriteJSON(os.Stdout, result); err != nil {
		return 4
	}
	if result.Outcome != "pass" {
		return 3
	}
	return 0
}
