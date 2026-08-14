// binparse reports structural metadata from ELF, Mach-O, and PE binaries.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/binparse"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("binparse", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	format := fs.String("format", "json", "json or text")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if *format != "json" && *format != "text" || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: binparse [--format json|text] <binary>")
		return cli.ExitUsage
	}
	report, err := binparse.Parse(fs.Args()[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitRuntime
	}
	if *format == "json" {
		if err := cli.WriteJSON(os.Stdout, report); err != nil {
			return cli.ExitRuntime
		}
		return cli.ExitOK
	}
	fmt.Fprintf(os.Stdout, "%s: %s (%s, %s)\nentry: %#x; sections: %d; symbols: %d\n", report.Path, report.Format, report.Architecture, report.ByteOrder, report.Entry, len(report.Sections), report.Symbols)
	for _, section := range report.Sections {
		fmt.Fprintf(os.Stdout, "  %-20s %#x %d %s\n", section.Name, section.Address, section.Size, section.Flags)
	}
	return cli.ExitOK
}
