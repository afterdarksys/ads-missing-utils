package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/envsub"
	"github.com/afterdarksys/ads-missing-utils/internal/varmerge"
)

type stringsFlag []string

func (v *stringsFlag) String() string         { return "" }
func (v *stringsFlag) Set(value string) error { *v = append(*v, value); return nil }
func main()                                   { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("varmerge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	schemaPath := fs.String("schema", "", "shared envsub YAML schema")
	envPrefix := fs.String("env-prefix", "", "process environment prefix")
	strict := fs.Bool("strict-schema", false, "reject keys not present in the schema")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	var files, envFiles, sets stringsFlag
	fs.Var(&files, "file", "JSON or YAML source (repeatable)")
	fs.Var(&envFiles, "env-file", "dotenv source (repeatable)")
	fs.Var(&sets, "set", "NAME=value source (repeatable)")
	args := cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--schema": true, "--env-prefix": true, "--file": true, "--env-file": true, "--set": true, "--format": true})
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if *format != "json" {
		fmt.Fprintln(os.Stderr, "--format must be json")
		return cli.ExitUsage
	}
	if len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "varmerge accepts source flags only")
		return cli.ExitUsage
	}
	schema, err := envsub.LoadSchema(*schemaPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	sources := []varmerge.Source{}
	for _, path := range files {
		source, err := varmerge.FileSource(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return cli.ExitCode(err)
		}
		sources = append(sources, source)
	}
	for _, path := range envFiles {
		source, err := varmerge.EnvFileSource(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return cli.ExitCode(err)
		}
		sources = append(sources, source)
	}
	if *envPrefix != "" {
		sources = append(sources, varmerge.EnvSource(*envPrefix))
	}
	if len(sets) > 0 {
		source, err := varmerge.SetSource(sets)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return cli.ExitCode(err)
		}
		sources = append(sources, source)
	}
	result, err := varmerge.Merge(sources, schema, *strict)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if err := cli.WriteJSON(os.Stdout, result); err != nil {
		return cli.ExitRuntime
	}
	return cli.ExitOK
}
