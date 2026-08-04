package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/jwalk"
)

func main() { os.Exit(run()) }

func run() int {
	fs := flag.NewFlagSet("jwalk", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	format := fs.String("format", "ndjson", "json or ndjson")
	include := fs.String("include", "", "regular expression to include")
	exclude := fs.String("exclude", "", "regular expression to exclude")
	types := fs.String("type", "", "comma-separated file,directory,symlink,other")
	minSize := fs.String("min-size", "", "minimum size")
	maxSize := fs.String("max-size", "", "maximum size")
	older := fs.String("older-than", "", "minimum age duration")
	newer := fs.String("newer-than", "", "maximum age duration")
	follow := fs.Bool("follow-symlinks", false, "include symlink entries")
	fail := fs.Bool("fail-on-error", false, "stop on traversal errors")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--format": true, "--include": true, "--exclude": true, "--type": true, "--min-size": true, "--max-size": true, "--older-than": true, "--newer-than": true})
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Println(cli.Version)
		return cli.ExitOK
	}
	if *format != "json" && *format != "ndjson" {
		fmt.Fprintln(os.Stderr, "--format must be json or ndjson")
		return cli.ExitUsage
	}
	opts := jwalk.Options{Roots: fs.Args(), Types: map[string]bool{}, FollowSymlinks: *follow, FailOnError: *fail}
	if *types != "" {
		for _, t := range strings.Split(*types, ",") {
			opts.Types[strings.TrimSpace(t)] = true
		}
	}
	var err error
	if *include != "" {
		opts.Include, err = regexp.Compile(*include)
		if err != nil {
			fmt.Fprintf(os.Stderr, "--include: %v\n", err)
			return cli.ExitUsage
		}
	}
	if *exclude != "" {
		opts.Exclude, err = regexp.Compile(*exclude)
		if err != nil {
			fmt.Fprintf(os.Stderr, "--exclude: %v\n", err)
			return cli.ExitUsage
		}
	}
	if *minSize != "" {
		v, e := jwalk.ParseSize(*minSize)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			return cli.ExitUsage
		}
		opts.MinSize = &v
	}
	if *maxSize != "" {
		v, e := jwalk.ParseSize(*maxSize)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			return cli.ExitUsage
		}
		opts.MaxSize = &v
	}
	if *older != "" {
		v, e := time.ParseDuration(*older)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			return cli.ExitUsage
		}
		opts.OlderThan = &v
	}
	if *newer != "" {
		v, e := time.ParseDuration(*newer)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			return cli.ExitUsage
		}
		opts.NewerThan = &v
	}
	records, err := jwalk.Walk(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if *format == "json" {
		if err := cli.WriteJSON(os.Stdout, records); err != nil {
			return cli.ExitRuntime
		}
	} else {
		for _, record := range records {
			if err := cli.WriteJSON(os.Stdout, record); err != nil {
				return cli.ExitRuntime
			}
		}
	}
	for _, record := range records {
		if record.Status == "error" {
			return cli.ExitPartial
		}
	}
	return cli.ExitOK
}
