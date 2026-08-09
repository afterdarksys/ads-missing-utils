package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/macossec"
	"io"
	"os"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("amtm", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	input := fs.String("input", "", "AppleScript file; stdin when omitted")
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
	var data []byte
	var err error
	if *input == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(*input)
	}
	if err != nil {
		return fail(err)
	}
	return write(macossec.MapAppleScriptIntent(string(data)))
}
func write(v any) int {
	if err := cli.WriteJSON(os.Stdout, v); err != nil {
		return 4
	}
	return 0
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return 4 }
