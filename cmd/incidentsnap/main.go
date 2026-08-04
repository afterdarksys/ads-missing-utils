package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"os"
	"runtime"
	"time"
)

func main() {
	fs := flag.NewFlagSet("incidentsnap", flag.ExitOnError)
	format := fs.String("format", "json", "json")
	if fs.Parse(os.Args[1:]) != nil || *format != "json" {
		os.Exit(2)
	}
	host, _ := os.Hostname()
	cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/incidentsnap/v1", "outcome": "partial", "captured_at": time.Now().UTC(), "host": host, "goos": runtime.GOOS, "goarch": runtime.GOARCH, "conclusion": "minimal host identity snapshot collected; collector selection and manifest persistence are pending"})
	fmt.Fprintln(os.Stderr, "incidentsnap: minimal read-only snapshot")
	os.Exit(3)
}
