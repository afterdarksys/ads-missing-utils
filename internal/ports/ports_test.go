package ports

import "testing"

func TestSocketAddress(t *testing.T) {
	address, port, err := socketAddress("0100007F:01BB", "ipv4")
	if err != nil || address != "127.0.0.1" || port != 443 {
		t.Fatalf("%s %d %v", address, port, err)
	}
}
