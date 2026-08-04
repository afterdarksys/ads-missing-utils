package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"os"
	"os/exec"
)

func main() {
	fs := flag.NewFlagSet("regocheck", flag.ExitOnError)
	policy := fs.String("policy", "", "Rego policy file or directory")
	format := fs.String("format", "json", "json")
	if fs.Parse(os.Args[1:]) != nil || *policy == "" || *format != "json" {
		os.Exit(2)
	}
	path, e := exec.LookPath("opa")
	if e != nil {
		cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/regocheck/v1", "outcome": "partial", "policy": *policy, "conclusion": "OPA executable is not installed; policy was not evaluated"})
		os.Exit(3)
	}
	version, e := exec.Command(path, "version", "--format=json").Output()
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(4)
	}
	cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/regocheck/v1", "outcome": "partial", "policy": *policy, "opa_version": string(version), "conclusion": "OPA engine detected; evaluation and fixture execution require an explicit decision/input contract"})
	os.Exit(3)
}
