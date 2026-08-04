package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("netwhy", "Explain DNS, routing, proxy, and firewall decisions for a connection.", os.Args[1:], os.Stdout, os.Stderr))
}
