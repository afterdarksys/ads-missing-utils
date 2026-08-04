package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/jsongate"
	"os"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("jsongate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	policyPath := fs.String("policy", "", "YAML gate policy")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--policy": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return 0
	}
	if *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--format must be json")
		return 2
	}
	policy, err := jsongate.LoadPolicy(*policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	findings, err := jsongate.Decode(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	result := jsongate.Evaluate(findings, policy)
	if err := cli.WriteJSON(os.Stdout, result); err != nil {
		return 4
	}
	if result.Decision == "deny" {
		return 1
	}
	return 0
}
