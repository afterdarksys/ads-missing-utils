package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"io"
	"os"
)

func main() {
	fs := flag.NewFlagSet("patchwhy", flag.ExitOnError)
	path := fs.String("path", "", "binary or library path")
	format := fs.String("format", "json", "json")
	if fs.Parse(os.Args[1:]) != nil || *path == "" || *format != "json" {
		os.Exit(2)
	}
	f, e := os.Open(*path)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(4)
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/patchwhy/v1", "outcome": "partial", "path": *path, "sha256": hex.EncodeToString(h.Sum(nil)), "conclusion": "current file identity collected; process mappings and restart recommendations are unavailable"})
	os.Exit(3)
}
