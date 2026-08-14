// Package pcapwhy turns offline packet captures into bounded network evidence.
package pcapwhy

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

const Schema = "missing-utils/pcapwhy/v1"

type Options struct{ MaxPackets, MaxFlows int }
type Report struct {
	Schema      string           `json:"schema"`
	Outcome     string           `json:"outcome"`
	Captures    []Capture        `json:"captures"`
	Comparisons []Comparison     `json:"comparisons,omitempty"`
	Diagnostics []cli.Diagnostic `json:"diagnostics,omitempty"`
	Conclusion  string           `json:"conclusion"`
}
type Capture struct {
	Path      string     `json:"path"`
	Packets   int        `json:"packets"`
	Truncated bool       `json:"truncated,omitempty"`
	Flows     []Flow     `json:"flows,omitempty"`
	VLANs     []uint16   `json:"vlans,omitempty"`
	MACs      []string   `json:"mac_addresses,omitempty"`
	ICMP      []ICMP     `json:"icmp,omitempty"`
	Resets    []TCPEvent `json:"tcp_resets,omitempty"`
	Payload   Payload    `json:"payload"`
}
type Flow struct {
	Source, Destination, Protocol string
	Packets, Bytes                int
}
type ICMP struct{ Source, Destination, Protocol, TypeCode, Meaning string }
type TCPEvent struct {
	Source, Destination string
	Flags               string
}
type Payload struct {
	Packets, Bytes, MaxBytes int
	Decoders                 []string `json:"decoders,omitempty"`
}
type Comparison struct {
	Flow   string   `json:"flow"`
	SeenIn []string `json:"seen_in"`
}

func Analyze(paths []string, options Options) (Report, error) {
	if len(paths) == 0 {
		return Report{}, cli.NewError(cli.ExitUsage, "at least one capture path is required")
	}
	if options.MaxPackets < 1 || options.MaxPackets > 1_000_000 {
		return Report{}, cli.NewError(cli.ExitUsage, "--max-packets must be between 1 and 1000000")
	}
	if options.MaxFlows < 1 || options.MaxFlows > 100_000 {
		return Report{}, cli.NewError(cli.ExitUsage, "--max-flows must be between 1 and 100000")
	}
	report := Report{Schema: Schema, Outcome: "pass", Captures: make([]Capture, 0, len(paths))}
	for _, path := range paths {
		capture, err := analyzeFile(path, options)
		if err != nil {
			return Report{}, err
		}
		report.Captures = append(report.Captures, capture)
		if capture.Truncated {
			report.Outcome = "partial"
		}
	}
	if len(report.Captures) > 1 {
		report.Comparisons = compare(report.Captures)
	}
	if report.Outcome == "partial" {
		report.Conclusion = "bounded packet evidence collected; one or more captures reached the configured packet limit"
	} else {
		report.Conclusion = "offline packet evidence collected; capture position, loss, encryption, NAT, and asymmetric routing can limit attribution"
	}
	return report, nil
}

func analyzeFile(path string, options Options) (Capture, error) {
	f, err := os.Open(path)
	if err != nil {
		return Capture{}, cli.NewError(cli.ExitRuntime, "open capture %s: %v", path, err)
	}
	defer f.Close()
	var reader interface {
		ReadPacketData() ([]byte, gopacket.CaptureInfo, error)
		LinkType() layers.LinkType
	}
	if strings.EqualFold(filepath.Ext(path), ".pcapng") {
		reader, err = pcapgo.NewNgReader(f, pcapgo.DefaultNgReaderOptions)
	} else {
		reader, err = pcapgo.NewReader(f)
	}
	if err != nil {
		return Capture{}, cli.NewError(cli.ExitRuntime, "read capture %s: %v", path, err)
	}
	c := Capture{Path: path, Flows: []Flow{}, VLANs: []uint16{}, MACs: []string{}, ICMP: []ICMP{}, Resets: []TCPEvent{}}
	flows, vlans, macs := map[string]*Flow{}, map[uint16]bool{}, map[string]bool{}
	icmp, resets, decoders := map[string]ICMP{}, map[string]TCPEvent{}, map[string]bool{}
	for c.Packets < options.MaxPackets {
		data, info, err := reader.ReadPacketData()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Capture{}, cli.NewError(cli.ExitRuntime, "read packet in %s: %v", path, err)
		}
		c.Packets++
		packet := gopacket.NewPacket(data, reader.LinkType(), gopacket.DecodeOptions{Lazy: true, NoCopy: true})
		if ethernet := packet.Layer(layers.LayerTypeEthernet); ethernet != nil {
			e := ethernet.(*layers.Ethernet)
			macs[e.SrcMAC.String()] = true
			macs[e.DstMAC.String()] = true
		}
		if vlan := packet.Layer(layers.LayerTypeDot1Q); vlan != nil {
			vlans[vlan.(*layers.Dot1Q).VLANIdentifier] = true
		}
		source, destination, protocol := endpoints(packet)
		if source != "" {
			key := source + " -> " + destination + " " + protocol
			flow := flows[key]
			if flow == nil {
				if len(flows) >= options.MaxFlows {
					continue
				}
				flow = &Flow{Source: source, Destination: destination, Protocol: protocol}
				flows[key] = flow
			}
			flow.Packets++
			flow.Bytes += info.Length
		}
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp := tcpLayer.(*layers.TCP)
			if tcp.RST {
				resets[source+" -> "+destination] = TCPEvent{Source: source, Destination: destination, Flags: tcpFlags(tcp)}
			}
		}
		if event, ok := icmpEvent(packet, source, destination); ok {
			icmp[event.Source+event.Destination+event.TypeCode] = event
		}
		if app := packet.ApplicationLayer(); app != nil && len(app.Payload()) > 0 {
			c.Payload.Packets++
			c.Payload.Bytes += len(app.Payload())
			if len(app.Payload()) > c.Payload.MaxBytes {
				c.Payload.MaxBytes = len(app.Payload())
			}
			for _, decoder := range applicationDecoders(packet) {
				decoders[decoder] = true
			}
		}
	}
	if c.Packets == options.MaxPackets {
		c.Truncated = true
	}
	for _, flow := range flows {
		c.Flows = append(c.Flows, *flow)
	}
	for vlan := range vlans {
		c.VLANs = append(c.VLANs, vlan)
	}
	for mac := range macs {
		c.MACs = append(c.MACs, mac)
	}
	for _, event := range icmp {
		c.ICMP = append(c.ICMP, event)
	}
	for _, event := range resets {
		c.Resets = append(c.Resets, event)
	}
	for decoder := range decoders {
		c.Payload.Decoders = append(c.Payload.Decoders, decoder)
	}
	sort.Slice(c.Flows, func(i, j int) bool {
		return c.Flows[i].Source+c.Flows[i].Destination+c.Flows[i].Protocol < c.Flows[j].Source+c.Flows[j].Destination+c.Flows[j].Protocol
	})
	sort.Slice(c.ICMP, func(i, j int) bool {
		return c.ICMP[i].Source+c.ICMP[i].Destination+c.ICMP[i].TypeCode < c.ICMP[j].Source+c.ICMP[j].Destination+c.ICMP[j].TypeCode
	})
	sort.Slice(c.Resets, func(i, j int) bool {
		return c.Resets[i].Source+c.Resets[i].Destination < c.Resets[j].Source+c.Resets[j].Destination
	})
	sort.Slice(c.VLANs, func(i, j int) bool { return c.VLANs[i] < c.VLANs[j] })
	sort.Strings(c.MACs)
	sort.Strings(c.Payload.Decoders)
	return c, nil
}

func endpoints(packet gopacket.Packet) (string, string, string) {
	var source, destination string
	if ipv4 := packet.Layer(layers.LayerTypeIPv4); ipv4 != nil {
		ip := ipv4.(*layers.IPv4)
		source, destination = ip.SrcIP.String(), ip.DstIP.String()
	}
	if ipv6 := packet.Layer(layers.LayerTypeIPv6); ipv6 != nil {
		ip := ipv6.(*layers.IPv6)
		source, destination = ip.SrcIP.String(), ip.DstIP.String()
	}
	if source == "" {
		return "", "", ""
	}
	if tcp := packet.Layer(layers.LayerTypeTCP); tcp != nil {
		t := tcp.(*layers.TCP)
		return net.JoinHostPort(source, fmt.Sprint(t.SrcPort)), net.JoinHostPort(destination, fmt.Sprint(t.DstPort)), "tcp"
	}
	if udp := packet.Layer(layers.LayerTypeUDP); udp != nil {
		u := udp.(*layers.UDP)
		return net.JoinHostPort(source, fmt.Sprint(u.SrcPort)), net.JoinHostPort(destination, fmt.Sprint(u.DstPort)), "udp"
	}
	if packet.Layer(layers.LayerTypeICMPv4) != nil {
		return source, destination, "icmp"
	}
	if packet.Layer(layers.LayerTypeICMPv6) != nil {
		return source, destination, "icmpv6"
	}
	return source, destination, "ip"
}

func icmpEvent(packet gopacket.Packet, source, destination string) (ICMP, bool) {
	if layer := packet.Layer(layers.LayerTypeICMPv4); layer != nil {
		i := layer.(*layers.ICMPv4)
		return ICMP{source, destination, "icmp", i.TypeCode.String(), icmpMeaning(i.TypeCode.String())}, true
	}
	if layer := packet.Layer(layers.LayerTypeICMPv6); layer != nil {
		i := layer.(*layers.ICMPv6)
		return ICMP{source, destination, "icmpv6", i.TypeCode.String(), icmpMeaning(i.TypeCode.String())}, true
	}
	return ICMP{}, false
}

func icmpMeaning(typeCode string) string {
	lower := strings.ToLower(typeCode)
	switch {
	case strings.Contains(lower, "administratively prohibited"):
		return "administratively prohibited; a filtering device may have rejected the traffic"
	case strings.Contains(lower, "fragmentation needed") || strings.Contains(lower, "packet too big"):
		return "path MTU signal; reduce packet size or inspect MTU configuration"
	case strings.Contains(lower, "destination unreachable"):
		return "destination unreachable; inspect routing, policy, and destination availability"
	case strings.Contains(lower, "time exceeded"):
		return "hop limit or TTL expired; inspect routing loops and path length"
	default:
		return "ICMP control or diagnostic message observed"
	}
}

func tcpFlags(tcp *layers.TCP) string {
	flags := []string{}
	if tcp.SYN {
		flags = append(flags, "SYN")
	}
	if tcp.ACK {
		flags = append(flags, "ACK")
	}
	if tcp.RST {
		flags = append(flags, "RST")
	}
	if tcp.FIN {
		flags = append(flags, "FIN")
	}
	return strings.Join(flags, ",")
}
func applicationDecoders(packet gopacket.Packet) []string {
	result := []string{}
	if packet.Layer(layers.LayerTypeDNS) != nil {
		result = append(result, "dns")
	}
	if packet.Layer(layers.LayerTypeTLS) != nil {
		result = append(result, "tls")
	}
	return result
}
func compare(captures []Capture) []Comparison {
	seen := map[string]map[string]bool{}
	for _, c := range captures {
		for _, f := range c.Flows {
			key := f.Source + " -> " + f.Destination + " " + f.Protocol
			if seen[key] == nil {
				seen[key] = map[string]bool{}
			}
			seen[key][c.Path] = true
		}
	}
	result := []Comparison{}
	for flow, paths := range seen {
		item := Comparison{Flow: flow}
		for path := range paths {
			item.SeenIn = append(item.SeenIn, path)
		}
		sort.Strings(item.SeenIn)
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Flow < result[j].Flow })
	return result
}
