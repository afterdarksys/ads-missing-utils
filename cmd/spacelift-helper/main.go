package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("spacelift-helper", "Normalize Spacelift hook context and suite reports.", os.Args[1:], os.Stdout, os.Stderr))
}
