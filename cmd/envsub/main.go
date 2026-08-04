package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/envsub"
)

type stringsFlag []string

func (v *stringsFlag) String() string     { return "" }
func (v *stringsFlag) Set(s string) error { *v = append(*v, s); return nil }

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("envsub", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	input := fs.String("input", "", "template input")
	output := fs.String("output", "", "destination file; stdout when omitted")
	schema := fs.String("schema", "", "YAML schema")
	strict := fs.Bool("strict-schema", false, "reject unknown keys from supplied sources")
	check := fs.Bool("check", false, "validate without rendering output")
	list := fs.Bool("list-keys", false, "list referenced keys")
	explain := fs.Bool("explain", false, "show redacted resolution sources")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	var envFiles, sets stringsFlag
	fs.Var(&envFiles, "env-file", "dotenv file (repeatable)")
	fs.Var(&sets, "set", "NAME=value (repeatable)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Println(cli.Version)
		return 0
	}
	if *format != "json" && *format != "ndjson" {
		fmt.Fprintln(os.Stderr, "--format must be json or ndjson")
		return cli.ExitUsage
	}
	result, err := envsub.Run(envsub.Options{Input: *input, Output: *output, EnvFiles: envFiles, Sets: sets, SchemaPath: *schema, StrictSchema: *strict, Check: *check, ListKeys: *list, Explain: *explain})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}
	if err := cli.WriteJSON(os.Stdout, result); err != nil {
		return cli.ExitRuntime
	}
	return cli.ExitOK
}
