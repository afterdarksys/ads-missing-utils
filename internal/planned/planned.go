// Package planned provides a transparent scaffold for roadmap commands.
//
// These binaries are built so packaging and integration tests can exercise the
// complete command inventory before each command's implementation is ready.
package planned

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// Run implements the common command contract for a planned utility.
func Run(name, summary string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.Bool("version", false, "print version")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [options]\n\n%s\n\n", name, summary)
		fmt.Fprintln(stderr, "Status: planned; this test scaffold has no operational implementation yet.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *version {
		fmt.Fprintln(stdout, "0.1.0-dev")
		return 0
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "%s: not implemented; see --help\n", name)
		return 1
	}
	fs.Usage()
	return 1
}
