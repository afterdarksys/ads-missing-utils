package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/macossec"
	"os"
	"os/exec"
	"path/filepath"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("aaas", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	db := fs.String("db", "", "TCC.db path")
	revoke := fs.String("revoke", "", "bundle identifier whose Accessibility grant should be reset")
	apply := fs.Bool("apply", false, "apply the requested revocation")
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
	if *revoke != "" {
		if !*apply {
			fmt.Fprintln(os.Stderr, "--revoke requires --apply")
			return 2
		}
		if err := exec.Command("tccutil", "reset", "Accessibility", *revoke).Run(); err != nil {
			return fail(err)
		}
	}
	home, _ := os.UserHomeDir()
	if *db == "" {
		*db = filepath.Join(home, "Library", "Application Support", "com.apple.TCC", "TCC.db")
	}
	grants, err := macossec.AuditAccessibility(*db)
	if err != nil {
		return fail(err)
	}
	return write(map[string]any{"schema": "missing-utils/aaas/v1", "grants": grants, "revoked": *revoke})
}
func write(v any) int {
	if err := cli.WriteJSON(os.Stdout, v); err != nil {
		return 4
	}
	return 0
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return 4 }
