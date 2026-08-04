package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"os"
	"os/user"
	"path/filepath"
)

type result struct {
	Schema     string `json:"schema"`
	Path       string `json:"path"`
	Outcome    string `json:"outcome"`
	Mode       string `json:"mode"`
	Identity   string `json:"identity"`
	Conclusion string `json:"conclusion"`
}

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("accesswhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("path", "", "filesystem path")
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
	if *path == "" || *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--path and --format json are required")
		return 2
	}
	info, err := os.Stat(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	identity := "unknown"
	if current, err := user.Current(); err == nil {
		identity = current.Username
	}
	outcome, conclusion := "pass", "mode bits are readable; ACL, mount, and LSM evidence is not yet collected"
	if info.Mode().Perm() == 0 {
		outcome = "partial"
		conclusion = "mode bits deny all access; ACL, mount, and LSM evidence is not yet collected"
	}
	if err := cli.WriteJSON(os.Stdout, result{"missing-utils/accesswhy/v1", filepath.Clean(*path), outcome, info.Mode().String(), identity, conclusion}); err != nil {
		return 4
	}
	if outcome == "partial" {
		return 3
	}
	return 0
}
