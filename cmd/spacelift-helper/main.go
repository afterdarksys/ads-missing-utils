package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"os"
)

func main() {
	fs := flag.NewFlagSet("spacelift-helper", flag.ExitOnError)
	format := fs.String("format", "json", "json")
	if fs.Parse(os.Args[1:]) != nil || *format != "json" || len(fs.Args()) > 1 {
		os.Exit(2)
	}
	values := map[string]string{}
	for _, key := range []string{"SPACELIFT_RUN_ID", "SPACELIFT_STACK_ID", "SPACELIFT_STAGE", "SPACELIFT_COMMIT_SHA"} {
		if value, ok := os.LookupEnv(key); ok {
			values[key] = value
		}
	}
	cli.WriteJSON(os.Stdout, map[string]any{"schema": "missing-utils/spacelift-helper/v1", "outcome": "partial", "context": values, "conclusion": "allowlisted Spacelift context collected; hook invocation and report retention are unavailable"})
	fmt.Fprintln(os.Stderr, "spacelift-helper: context mode")
	os.Exit(3)
}
