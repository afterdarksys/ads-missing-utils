package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/homestate"
)

type outputData struct {
	Root        string                   `json:"root"`
	State       string                   `json:"state,omitempty"`
	EntryCount  int                      `json:"entry_count,omitempty"`
	Changes     []homestate.Change       `json:"changes,omitempty"`
	BrokenNames []homestate.BrokenName   `json:"broken_names,omitempty"`
	Renames     []homestate.RenameRecord `json:"renames,omitempty"`
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	fs := flag.NewFlagSet("homestate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	index := fs.Bool("index", false, "index the mapped directory")
	check := fs.Bool("check", false, "compare disk with indexed state")
	broken := fs.Bool("broken-file-names", false, "find problematic path components")
	newFiles := fs.Bool("new-files", false, "show files added since indexing")
	movedFiles := fs.Bool("moved-files", false, "show files likely moved since indexing")
	fixNames := fs.Bool("fix-names", false, "repair problematic names and save reversible mappings")
	restoreNames := fs.Bool("restore-names", false, "restore names previously repaired")
	mapDir := fs.String("map-dir", "", "directory to map instead of $HOME")
	state := fs.String("state", "", "state database path")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cli.ExitOK
		}
		return cli.ExitUsage
	}
	if *version {
		fmt.Println(cli.Version)
		return cli.ExitOK
	}
	modes := []bool{*index, *check, *broken, *newFiles, *movedFiles, *fixNames, *restoreNames}
	count := 0
	for _, enabled := range modes {
		if enabled {
			count++
		}
	}
	if count != 1 {
		fmt.Fprintln(os.Stderr, "choose exactly one operation: --index, --check, --broken-file-names, --new-files, --moved-files, --fix-names, or --restore-names")
		return cli.ExitUsage
	}
	root := *mapDir
	if root == "" {
		var err error
		root, err = os.UserHomeDir()
		if err != nil || root == "" {
			fmt.Fprintln(os.Stderr, "cannot determine $HOME; use --map-dir")
			return cli.ExitUsage
		}
	}
	root, _ = filepath.Abs(root)
	if *state == "" {
		*state = filepath.Join(root, ".local", "state", "missing-utils", "homestate.json")
	}
	data := outputData{Root: root, State: *state}
	outcome := "pass"
	diagnostics := []cli.Diagnostic{}
	if *index {
		db, err := homestate.Index(root, *state)
		if err != nil {
			return fail(err)
		}
		data.EntryCount = len(db.Entries)
	} else if *broken {
		items, err := homestate.BrokenNames(root)
		if err != nil {
			return fail(err)
		}
		data.BrokenNames = items
		if len(items) > 0 {
			outcome = "changes"
		}
	} else if *fixNames {
		items, diags, err := homestate.FixNames(root, *state)
		if err != nil {
			return fail(err)
		}
		data.Renames = items
		diagnostics = convertDiagnostics(diags)
		if len(items) > 0 {
			outcome = "changes"
		}
	} else if *restoreNames {
		items, diags, err := homestate.RestoreNames(root, *state)
		if err != nil {
			return fail(err)
		}
		data.Renames = items
		diagnostics = convertDiagnostics(diags)
		if len(items) > 0 {
			outcome = "changes"
		}
	} else {
		db, err := homestate.Load(*state)
		if err != nil {
			return fail(err)
		}
		changes, diags, err := homestate.Check(db, *state)
		if err != nil {
			return fail(err)
		}
		diagnostics = convertDiagnostics(diags)
		for _, change := range changes {
			if *newFiles && change.Status != "added" {
				continue
			}
			if *movedFiles && change.Status != "moved" && change.Status != "ambiguous_move" {
				continue
			}
			data.Changes = append(data.Changes, change)
		}
		if len(data.Changes) > 0 {
			outcome = "changes"
		}
	}
	if len(diagnostics) > 0 && outcome == "pass" {
		outcome = "partial"
	}
	if err := cli.WriteJSON(os.Stdout, cli.Response[outputData]{Schema: "missing-utils/homestate/v1", Command: "homestate", Outcome: outcome, Data: data, Diagnostics: diagnostics}); err != nil {
		return cli.ExitRuntime
	}
	if len(diagnostics) > 0 {
		return cli.ExitPartial
	}
	return cli.ExitOK
}

func convertDiagnostics(values []homestate.Diagnostic) []cli.Diagnostic {
	out := make([]cli.Diagnostic, 0, len(values))
	for _, value := range values {
		out = append(out, cli.Diagnostic{Code: value.Code, Message: value.Message, Path: value.Path})
	}
	return out
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return cli.ExitRuntime }
