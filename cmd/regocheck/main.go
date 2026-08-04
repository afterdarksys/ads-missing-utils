package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("regocheck", "Evaluate and test Rego policies against versioned fixtures.", os.Args[1:], os.Stdout, os.Stderr))
}
