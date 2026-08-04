package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("servicewhy", "Explain a service's health, restart, or dependency state.", os.Args[1:], os.Stdout, os.Stderr))
}
