package pcapwhy

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func TestAnalyzeCapturesVLANFlowAndReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.pcap")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := pcapgo.NewWriter(f)
	if err := writer.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
		t.Fatal(err)
	}
	buffer := gopacket.NewSerializeBuffer()
	ip := &layers.IPv4{Version: 4, TTL: 64, SrcIP: []byte{10, 0, 0, 1}, DstIP: []byte{10, 0, 0, 2}, Protocol: layers.IPProtocolTCP}
	tcp := &layers.TCP{SrcPort: 443, DstPort: 51000, RST: true, ACK: true}
	if err := tcp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	err = gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{SrcMAC: []byte{0, 1, 2, 3, 4, 5}, DstMAC: []byte{6, 7, 8, 9, 10, 11}, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: 42, Type: layers.EthernetTypeIPv4},
		ip,
		tcp,
		gopacket.Payload([]byte("opaque-payload")))
	if err != nil {
		t.Fatal(err)
	}
	data := buffer.Bytes()
	if err := writer.WritePacket(gopacket.CaptureInfo{Timestamp: time.Now(), CaptureLength: len(data), Length: len(data)}, data); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	report, err := Analyze([]string{path}, Options{MaxPackets: 10, MaxFlows: 10})
	if err != nil {
		t.Fatal(err)
	}
	capture := report.Captures[0]
	if capture.Packets != 1 || len(capture.Flows) != 1 || len(capture.Resets) != 1 {
		t.Fatalf("unexpected capture summary: %#v", capture)
	}
	if len(capture.VLANs) != 1 || capture.VLANs[0] != 42 {
		t.Fatalf("VLANs = %#v", capture.VLANs)
	}
	if capture.Payload.Bytes != len("opaque-payload") {
		t.Fatalf("payload = %#v", capture.Payload)
	}
}

func TestICMPMeaning(t *testing.T) {
	if got := icmpMeaning("Destination Unreachable (Administratively Prohibited)"); got == "ICMP control or diagnostic message observed" {
		t.Fatal("expected a specific ICMP explanation")
	}
}
