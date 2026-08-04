package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"io"
	"os"
	"path/filepath"
)

type result struct {
	Schema  string `json:"schema"`
	Outcome string `json:"outcome"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	SHA256  string `json:"sha256"`
}

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("binarywhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("path", "", "executable path")
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
	file, err := os.Open(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	if !info.Mode().IsRegular() {
		fmt.Fprintln(os.Stderr, "path is not a regular file")
		return 2
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	if err := cli.WriteJSON(os.Stdout, result{"missing-utils/binarywhy/v1", "pass", filepath.Clean(*path), info.Size(), info.Mode().String(), hex.EncodeToString(hash.Sum(nil))}); err != nil {
		return 4
	}
	return 0
}
