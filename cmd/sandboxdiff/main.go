package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("sandboxdiff", "Compare workload isolation and attack-surface characteristics.", os.Args[1:], os.Stdout, os.Stderr))
}
