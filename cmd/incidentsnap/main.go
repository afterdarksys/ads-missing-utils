package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("incidentsnap", "Collect a redacted, integrity-verifiable host snapshot.", os.Args[1:], os.Stdout, os.Stderr))
}
