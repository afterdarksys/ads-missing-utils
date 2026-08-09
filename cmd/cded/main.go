package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/macossec"
	"os"
	"os/exec"
	"strings"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("cded", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	input := fs.String("input", "", "inspect a file instead of clipboard")
	drain := fs.Bool("drain", false, "clear sensitive clipboard content")
	allow := fs.String("allow-process", "", "frontmost application name allowed to retain sensitive content")
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
	var content []byte
	var err error
	frontmost := ""
	if *input != "" {
		content, err = os.ReadFile(*input)
	} else {
		content, err = exec.Command("pbpaste").Output()
		if err == nil {
			out, _ := exec.Command("osascript", "-e", `tell application "System Events" to get name of first application process whose frontmost is true`).Output()
			frontmost = strings.TrimSpace(string(out))
		}
	}
	if err != nil {
		return fail(err)
	}
	matches := macossec.DetectSecrets(string(content))
	drained := false
	if *drain && *input == "" && len(matches) > 0 && !strings.EqualFold(frontmost, *allow) {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader("")
		if err := cmd.Run(); err != nil {
			return fail(err)
		}
		drained = true
	}
	return write(map[string]any{"schema": "missing-utils/cded/v1", "matches": matches, "frontmost_application": frontmost, "drained": drained})
}
func write(v any) int {
	if err := cli.WriteJSON(os.Stdout, v); err != nil {
		return 4
	}
	return 0
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return 4 }
