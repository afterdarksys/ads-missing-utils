package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/metascore"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: metascore index|check [options]")
		return cli.ExitUsage
	}
	if args[0] == "--help" || args[0] == "-h" {
		fmt.Println("usage: metascore index|check [--root DIR] [--state FILE]")
		return cli.ExitOK
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Println(cli.Version)
		return 0
	}
	fs := flag.NewFlagSet("metascore "+args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", "", "directory to snapshot")
	state := fs.String("state", "", "metadata state path")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return cli.ExitUsage
	}
	home, _ := os.UserHomeDir()
	if *root == "" {
		*root = home
	}
	if *state == "" {
		*state = filepath.Join(home, ".local", "state", "missing-utils", "metascore.json")
	}
	switch args[0] {
	case "index":
		snapshot, err := metascore.Index(*root, *state)
		if err != nil {
			return fail(err)
		}
		return write(map[string]any{"schema": "missing-utils/metascore/v1", "outcome": "pass", "root": snapshot.Root, "state": *state, "entry_count": len(snapshot.Entries)})
	case "check":
		report, err := metascore.Check(*state)
		if err != nil {
			return fail(err)
		}
		return write(report)
	default:
		fmt.Fprintln(os.Stderr, "usage: metascore index|check [options]")
		return cli.ExitUsage
	}
}
func write(value any) int {
	if err := cli.WriteJSON(os.Stdout, value); err != nil {
		return cli.ExitRuntime
	}
	return 0
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return cli.ExitRuntime }
