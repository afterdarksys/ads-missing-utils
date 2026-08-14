package netwhy

import "testing"

func TestParseGateway(t *testing.T) {
	got, err := parseGateway("0102A8C0")
	if err != nil || got != "192.168.2.1" {
		t.Fatalf("parseGateway() = %q, %v", got, err)
	}
}

func TestRedactProxy(t *testing.T) {
	if got := redactProxy("http://user:secret@proxy.example:8080/path?q=secret"); got != "http://proxy.example:8080" {
		t.Fatalf("redactProxy() = %q", got)
	}
}
