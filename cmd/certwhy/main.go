package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

type result struct {
	Schema          string   `json:"schema"`
	Outcome         string   `json:"outcome"`
	Address         string   `json:"address"`
	ServerName      string   `json:"server_name"`
	Subject         string   `json:"subject,omitempty"`
	Issuer          string   `json:"issuer,omitempty"`
	NotBefore       string   `json:"not_before,omitempty"`
	NotAfter        string   `json:"not_after,omitempty"`
	DNSNames        []string `json:"dns_names,omitempty"`
	SHA256          string   `json:"sha256,omitempty"`
	TLSVersion      string   `json:"tls_version,omitempty"`
	CipherSuite     string   `json:"cipher_suite,omitempty"`
	ValidationError string   `json:"validation_error,omitempty"`
	Conclusion      string   `json:"conclusion"`
}

func main() { os.Exit(run()) }

func run() int {
	fs := flag.NewFlagSet("certwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	address := fs.String("address", "", "TLS host:port")
	servername := fs.String("servername", "", "TLS server name (defaults to address host)")
	format := fs.String("format", "json", "output format (json)")
	version := fs.Bool("version", false, "print version")
	_ = fs.Bool("no-color", false, "disable color")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return cli.ExitUsage
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return cli.ExitOK
	}
	host, _, err := net.SplitHostPort(*address)
	if *address == "" || err != nil || *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--address must be a valid HOST:PORT and --format must be json")
		return cli.ExitUsage
	}
	if *servername != "" {
		host = *servername
	}
	report := result{Schema: "missing-utils/certwhy/v1", Address: *address, ServerName: host}
	conn, err := tls.Dial("tcp", *address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err == nil {
		defer conn.Close()
		fill(&report, conn.ConnectionState())
		report.Outcome = "pass"
		report.Conclusion = "TLS handshake and peer certificate chain validated"
		_ = cli.WriteJSON(os.Stdout, report)
		return cli.ExitOK
	}

	// A second, explicitly unverified connection is only used to describe the
	// offered certificate after verification failed. It never changes the result.
	report.ValidationError = err.Error()
	unverified, inspectErr := tls.Dial("tcp", *address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}) // #nosec G402 -- diagnostic evidence after a failed verified handshake
	if inspectErr != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitFailure
	}
	defer unverified.Close()
	fill(&report, unverified.ConnectionState())
	report.Outcome = "fail"
	report.Conclusion = "TLS validation failed; certificate details were collected from an unverified diagnostic connection"
	_ = cli.WriteJSON(os.Stdout, report)
	return cli.ExitFailure
}

func fill(report *result, state tls.ConnectionState) {
	if len(state.PeerCertificates) == 0 {
		return
	}
	certificate := state.PeerCertificates[0]
	fingerprint := sha256.Sum256(certificate.Raw)
	report.Subject, report.Issuer = certificate.Subject.String(), certificate.Issuer.String()
	report.NotBefore, report.NotAfter = certificate.NotBefore.UTC().Format(time.RFC3339), certificate.NotAfter.UTC().Format(time.RFC3339)
	report.DNSNames = append([]string(nil), certificate.DNSNames...)
	sort.Strings(report.DNSNames)
	report.SHA256 = hex.EncodeToString(fingerprint[:])
	report.TLSVersion = tls.VersionName(state.Version)
	report.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
}
