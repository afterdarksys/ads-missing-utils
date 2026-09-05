package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/runreceipt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("runreceipt", flag.ContinueOnError)
	receipt := fs.String("receipt", "", "new private receipt file; required with --execute")
	label := fs.String("label", "", "short operation label; do not include secrets")
	spec := fs.String("pipeline", "", "JSON array of argv arrays; no shell expansion")
	dir := fs.String("directory", ".", "working directory")
	execute := fs.Bool("execute", false, "execute explicitly supplied commands; default is preview")
	timeout := fs.Duration("timeout", 30*time.Minute, "local process deadline; maximum 24h")
	version := fs.Bool("version", false, "print version")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *version {
		fmt.Println(cli.Version)
		return 0
	}
	if strings.TrimSpace(*label) == "" || len(*label) > 256 || strings.ContainsFunc(*label, unicode.IsControl) || *timeout <= 0 || *timeout > 24*time.Hour {
		fmt.Fprintln(os.Stderr, "provide a single-line --label and valid timeout")
		return 2
	}
	directory, err := filepath.Abs(*dir)
	if err != nil {
		return 2
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		fmt.Fprintln(os.Stderr, "working directory is unavailable")
		return 2
	}
	var commands [][]string
	if *spec != "" {
		if fs.NArg() != 0 {
			fmt.Fprintln(os.Stderr, "--pipeline cannot be combined with command arguments")
			return 2
		}
		file, err := os.Open(*spec)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot read pipeline")
			return 2
		}
		data, err := io.ReadAll(io.LimitReader(file, 1<<20+1))
		file.Close()
		if err != nil || len(data) > 1<<20 || json.Unmarshal(data, &commands) != nil {
			fmt.Fprintln(os.Stderr, "pipeline must be a JSON array of argv arrays, at most 1 MiB")
			return 2
		}
	} else {
		commands = [][]string{fs.Args()}
	}
	preview, err := runreceipt.Preview(commands, *label, directory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if !*execute {
		if cli.WriteJSON(os.Stdout, preview) != nil {
			return 4
		}
		return 0
	}
	if *receipt == "" {
		fmt.Fprintln(os.Stderr, "--execute requires --receipt")
		return 2
	}
	first := true
	save := func(r runreceipt.Receipt) error {
		var data bytes.Buffer
		if err := json.NewEncoder(&data).Encode(r); err != nil {
			return err
		}
		if first {
			f, err := os.OpenFile(*receipt, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return fmt.Errorf("cannot create receipt; choose a new path in an existing directory")
			}
			_, err = f.Write(data.Bytes())
			if err == nil {
				err = f.Sync()
			}
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
			first = false
			return nil
		}
		f, err := os.CreateTemp(filepath.Dir(*receipt), ".runreceipt-*")
		if err != nil {
			return err
		}
		name := f.Name()
		defer os.Remove(name)
		if _, err = f.Write(data.Bytes()); err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		return os.Rename(name, *receipt)
	}
	base, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(base, *timeout)
	defer cancel()
	result, err := runreceipt.Execute(ctx, commands, *label, directory, os.Stdin, os.Stdout, os.Stderr, save)
	if err != nil {
		fmt.Fprintln(os.Stderr, "receipt recording failed; operation outcome must be checked independently")
		return 4
	}
	return runreceipt.ExitCode(result)
}
