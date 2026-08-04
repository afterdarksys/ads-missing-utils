package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("driftwhy", "Explain security-relevant state changes and their mechanisms.", os.Args[1:], os.Stdout, os.Stderr))
}
