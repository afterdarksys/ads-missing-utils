package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("patchwhy", "Identify processes using replaced code and required restarts.", os.Args[1:], os.Stdout, os.Stderr))
}
