package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("expose", "Report service reachability by interface and namespace.", os.Args[1:], os.Stdout, os.Stderr))
}
