package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/macossec"
	"os"
	"time"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	fs := flag.NewFlagSet("tjt", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	duration := fs.Duration("duration", time.Second, "observation duration")
	interval := fs.Duration("interval", 50*time.Millisecond, "sampling interval")
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
	if *duration <= 0 || *interval <= 0 {
		fmt.Fprintln(os.Stderr, "duration and interval must be positive")
		return 2
	}
	tracker := macossec.NewTracker()
	jobs := []macossec.TransientJob{}
	deadline := time.Now().Add(*duration)
	for {
		processes, err := macossec.SnapshotProcesses()
		if err != nil {
			return fail(err)
		}
		jobs = append(jobs, tracker.Observe(time.Now(), processes)...)
		if time.Now().Add(*interval).After(deadline) {
			break
		}
		time.Sleep(*interval)
	}
	return write(map[string]any{"schema": "missing-utils/tjt/v1", "interval": interval.String(), "jobs": jobs})
}
func write(v any) int {
	if err := cli.WriteJSON(os.Stdout, v); err != nil {
		return 4
	}
	return 0
}
func fail(err error) int { fmt.Fprintln(os.Stderr, err); return 4 }
