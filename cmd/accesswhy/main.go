package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("accesswhy", "Explain effective filesystem access for an identity.", os.Args[1:], os.Stdout, os.Stderr))
}
