package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/jsondiff"
	"os"
)

func main() {
	fs := flag.NewFlagSet("sandboxdiff", flag.ExitOnError)
	left := fs.String("left", "", "left JSON snapshot")
	right := fs.String("right", "", "right JSON snapshot")
	format := fs.String("format", "json", "json")
	if fs.Parse(os.Args[1:]) != nil || *left == "" || *right == "" || *format != "json" {
		os.Exit(2)
	}
	a, e := jsondiff.DecodeFile(*left)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(cli.ExitCode(e))
	}
	b, e := jsondiff.DecodeFile(*right)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(cli.ExitCode(e))
	}
	cli.WriteJSON(os.Stdout, jsondiff.Compare(a, b, jsondiff.Options{}))
}
