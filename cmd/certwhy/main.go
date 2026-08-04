package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("certwhy", "Explain certificate selection, trust, and validation failures.", os.Args[1:], os.Stdout, os.Stderr))
}
