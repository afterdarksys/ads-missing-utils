// Package netwhy collects bounded, local network troubleshooting evidence.
package netwhy

import (
	"bufio"
	"context"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
)

const Schema = "missing-utils/netwhy/v1"

type Report struct {
	Schema       string            `json:"schema"`
	Outcome      string            `json:"outcome"`
	Host         string            `json:"host"`
	Port         int               `json:"port,omitempty"`
	Addresses    []string          `json:"addresses,omitempty"`
	DefaultRoute string            `json:"default_route,omitempty"`
	Proxy        map[string]string `json:"proxy,omitempty"`
	Diagnostics  []string          `json:"diagnostics,omitempty"`
	Conclusion   string            `json:"conclusion"`
}

var lookupIP = net.LookupIP
var readFile = os.ReadFile

func Inspect(ctx context.Context, host string, port int) (Report, error) {
	_ = ctx // net.Resolver uses the system resolver and does not accept a context here.
	report := Report{Schema: Schema, Outcome: "partial", Host: host, Port: port, Proxy: proxyEnvironment()}
	ips, err := lookupIP(host)
	if err != nil {
		return report, err
	}
	for _, ip := range ips {
		report.Addresses = append(report.Addresses, ip.String())
	}
	sort.Strings(report.Addresses)
	if route, err := defaultRoute(); err == nil {
		report.DefaultRoute = route
	} else {
		report.Diagnostics = append(report.Diagnostics, "default route unavailable: "+err.Error())
	}
	report.Conclusion = "DNS and local route/proxy evidence collected; firewall, network namespace, and remote reachability are not inferred"
	return report, nil
}

func defaultRoute() (string, error) {
	data, err := readFile("/proc/net/route")
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	if !scanner.Scan() {
		return "", scanner.Err()
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || fields[1] != "00000000" {
			continue
		}
		gateway, err := parseGateway(fields[2])
		if err != nil {
			continue
		}
		return fields[0] + " via " + gateway, nil
	}
	return "", scanner.Err()
}

func parseGateway(value string) (string, error) {
	if len(value) != 8 {
		return "", &net.ParseError{Type: "IPv4 gateway", Text: value}
	}
	var b [4]byte
	for i := range b {
		part, err := parseHexByte(value[i*2 : i*2+2])
		if err != nil {
			return "", err
		}
		b[3-i] = part
	}
	return net.IP(b[:]).String(), nil
}

func parseHexByte(value string) (byte, error) {
	var n byte
	for _, r := range value {
		n <<= 4
		switch {
		case r >= '0' && r <= '9':
			n += byte(r - '0')
		case r >= 'a' && r <= 'f':
			n += byte(r-'a') + 10
		case r >= 'A' && r <= 'F':
			n += byte(r-'A') + 10
		default:
			return 0, &net.ParseError{Type: "hex", Text: value}
		}
	}
	return n, nil
}

func proxyEnvironment() map[string]string {
	proxies := map[string]string{}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"} {
		if value := os.Getenv(key); value != "" {
			proxies[key] = redactProxy(value)
		}
	}
	return proxies
}

func redactProxy(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "configured"
	}
	return parsed.Scheme + "://" + parsed.Host
}
