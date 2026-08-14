package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/logic"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("info2logic", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "", "Logic output path (default: stdout)")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--output": true})); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: info2logic [--output FILE] <info-source>")
		return cli.ExitUsage
	}
	in, err := os.Open(fs.Args()[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitRuntime
	}
	defer in.Close()
	d, err := logic.FromInfo(in, fs.Args()[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitUsage
	}
	return write(*output, d)
}
func write(path string, d logic.Document) int {
	var out io.Writer = os.Stdout
	var file *os.File
	var err error
	if path != "" {
		file, err = os.Create(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return cli.ExitRuntime
		}
		defer file.Close()
		out = file
	}
	if err := logic.Encode(out, d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitRuntime
	}
	return cli.ExitOK
}
