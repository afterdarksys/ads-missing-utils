// logic reads Logic, roff man, and GNU Info source files as one manual model.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/logic"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("logic", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	sourceType := fs.String("type", "auto", "auto, logic, man, or info")
	format := fs.String("format", "text", "text or json")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(cli.ReorderInterspersed(os.Args[1:], map[string]bool{"--type": true, "--format": true})); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	if len(fs.Args()) != 1 || (*format != "text" && *format != "json") {
		fmt.Fprintln(os.Stderr, "usage: logic [--type auto|logic|man|info] [--format text|json] <source>")
		return cli.ExitUsage
	}
	path := fs.Args()[0]
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitRuntime
	}
	defer file.Close()
	typeName := *sourceType
	if typeName == "auto" {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".logic":
			typeName = "logic"
		case ".info":
			typeName = "info"
		default:
			typeName = "man"
		}
	}
	var d logic.Document
	switch typeName {
	case "logic":
		d, err = logic.Decode(file)
	case "man":
		d, err = logic.FromMan(file, path)
	case "info":
		d, err = logic.FromInfo(file, path)
	default:
		err = fmt.Errorf("--type must be auto, logic, man, or info")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitUsage
	}
	if *format == "json" {
		if err := cli.WriteJSON(os.Stdout, d); err != nil {
			return cli.ExitRuntime
		}
		return cli.ExitOK
	}
	fmt.Fprintf(os.Stdout, "%s%s\n", d.Title, sectionSuffix(d.Section))
	if d.Summary != "" {
		fmt.Fprintln(os.Stdout, d.Summary)
	}
	for _, section := range d.Sections {
		fmt.Fprintf(os.Stdout, "\n%s\n%s\n", strings.ToUpper(section.Name), section.Body)
	}
	return cli.ExitOK
}
func sectionSuffix(section string) string {
	if section == "" {
		return ""
	}
	return " (" + section + ")"
}
