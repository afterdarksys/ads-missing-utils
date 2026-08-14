# pcapwhy

`pcapwhy` is an offline packet-capture evidence collector for incident response
and network troubleshooting. It does not replace Wireshark or a packet broker;
it produces bounded JSON suitable for tickets, support bundles, and automation.

```sh
pcapwhy inside.pcapng outside.pcapng --format json
pcapwhy --max-packets 250000 firewall-span.pcapng
```

It reports directional IPv4/IPv6 flows, observed Ethernet source and
destination MAC addresses, 802.1Q VLAN identifiers, TCP RST packets, and ICMP
or ICMPv6 type/code descriptions. Payload reporting contains only byte counts,
largest payload size, and recognized safe protocol decoders such as DNS or TLS;
it never outputs payload bytes.

When more than one capture is supplied, the `comparisons` field states which
input captures saw each directional flow. A SYN seen inside a firewall SPAN
capture but absent outside is useful evidence of a break at or before the
boundary. It is not conclusive proof: capture loss, aggregation, NAT,
asymmetric routing, and mirrored-port configuration can create the same
observation.

The tool is offline-only. It does not open interfaces, inject traffic, decode
credentials, or determine remote ACL policy. Treat capture files and JSON
output as sensitive because MAC addresses, IP addresses, and topology are often
operationally sensitive.
