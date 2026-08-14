// osx provides deliberate, scriptable access to common macOS subsystems.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/osx"
)

const usage = `Usage: osx [--apply] [--duration DURATION] <subsystem> <action> [arguments]

Read-only commands: info show; battery status; power settings; network quality|interfaces;
spotlight search QUERY; dns status; security filevault|gatekeeper|firewall|sip; updates list;
defaults read; clipboard read; volume get.
Actions requiring --apply: caffeinate start; launch open; dns flush; updates install;
defaults write|delete; speech say; clipboard write; files copy; automation run; volume set|mute|unmute.

Examples:
  osx defaults read com.apple.finder AppleShowAllFiles
  osx defaults write com.apple.finder AppleShowAllFiles bool true --apply
  osx caffeinate start --duration 45m --apply
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, osx.SystemRunner{})) }

func run(args []string, stdout, stderr io.Writer, runner osx.Runner) int {
	fs := flag.NewFlagSet("osx", flag.ContinueOnError)
	fs.SetOutput(stderr)
	apply := fs.Bool("apply", false, "allow a state-changing action")
	duration := fs.Duration("duration", 0, "caffeinate duration, for example 45m")
	version := fs.Bool("version", false, "print version")
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	if err := fs.Parse(cli.ReorderInterspersed(args, map[string]bool{"--duration": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cli.ExitOK
		}
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(stdout, cli.Version)
		return cli.ExitOK
	}
	if *duration < 0 {
		fmt.Fprintln(stderr, "--duration cannot be negative")
		return cli.ExitUsage
	}
	positionals := fs.Args()
	if len(positionals) == 1 && positionals[0] == "help" {
		fs.Usage()
		return cli.ExitOK
	}
	if len(positionals) < 2 {
		fs.Usage()
		return cli.ExitUsage
	}
	result, err := osx.Execute(context.Background(), runner, osx.Request{Subsystem: positionals[0], Action: positionals[1], Args: positionals[2:], Apply: *apply, Duration: *duration})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return cli.ExitRuntime
	}
	if err := cli.WriteJSON(stdout, result); err != nil {
		return cli.ExitRuntime
	}
	return cli.ExitOK
}
