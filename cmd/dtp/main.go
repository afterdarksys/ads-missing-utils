package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/macossec"
	"os"
	"path/filepath"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("dtp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	index := fs.Bool("index", false, "create signed baseline")
	check := fs.Bool("check", false, "verify and compare baseline")
	root := fs.String("root", "/Library/LaunchDaemons", "daemon executable root")
	state := fs.String("state", "", "baseline path")
	keyPath := fs.String("key", "", "HMAC key file (minimum 16 bytes)")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *version {
		fmt.Println(cli.Version)
		return 0
	}
	if *index == *check || *keyPath == "" {
		fmt.Fprintln(os.Stderr, "choose one of --index/--check and provide --key")
		return 2
	}
	home, _ := os.UserHomeDir()
	if *state == "" {
		*state = filepath.Join(home, ".local", "state", "missing-utils", "dtp.json")
	}
	key, err := os.ReadFile(*keyPath)
	if err != nil {
		return fail(err)
	}
	if *index {
		baseline, err := macossec.CreateDaemonBaseline(*root, key)
		if err != nil {
			return fail(err)
		}
		if err := macossec.SaveBaseline(*state, baseline); err != nil {
			return fail(err)
		}
		return write(map[string]any{"schema": "missing-utils/dtp/v1", "outcome": "pass", "entries": len(baseline.Entries), "scan_errors": baseline.ScanErrors, "state": *state})
	}
	baseline, err := macossec.LoadBaseline(*state)
	if err != nil {
		return fail(err)
	}
	changes, err := macossec.CheckDaemonBaseline(baseline, key)
	if err != nil {
		return fail(err)
	}
	return write(map[string]any{"schema": "missing-utils/dtp/v1", "outcome": map[bool]string{true: "changes", false: "pass"}[len(changes) > 0], "changes": changes})
}
func write(v any) int {
	if err := cli.WriteJSON(os.Stdout, v); err != nil {
		return 4
	}
	return 0
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return 4 }
