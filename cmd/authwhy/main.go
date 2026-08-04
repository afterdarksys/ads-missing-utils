package main

import (
	"os"

	"github.com/afterdarksys/ads-missing-utils/internal/planned"
)

func main() {
	os.Exit(planned.Run("authwhy", "Explain account, SSH, PAM, NSS, and policy effects on login.", os.Args[1:], os.Stdout, os.Stderr))
}
