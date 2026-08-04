// Package ports enumerates Linux TCP/UDP listeners from /proc without parsing
// human-oriented command output.
package ports

import (
	"bufio"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const Schema = "missing-utils/ports/v1"

type Record struct {
	Schema       string `json:"schema"`
	Status       string `json:"status"`
	Transport    string `json:"transport"`
	Family       string `json:"family"`
	LocalAddress string `json:"local_address"`
	LocalPort    int    `json:"local_port"`
	State        string `json:"state"`
	PID          int    `json:"pid,omitempty"`
	Process      string `json:"process,omitempty"`
	UID          int    `json:"uid,omitempty"`
	Visibility   string `json:"visibility"`
}
type Options struct {
	Port      int
	Transport string
}

func List(options Options) ([]Record, error) {
	if runtime.GOOS != "linux" {
		return nil, cli.NewError(cli.ExitRuntime, "ports is currently implemented for Linux")
	}
	owners := owners()
	records := []Record{}
	for _, source := range []struct{ path, transport, family string }{{"/proc/net/tcp", "tcp", "ipv4"}, {"/proc/net/tcp6", "tcp", "ipv6"}, {"/proc/net/udp", "udp", "ipv4"}, {"/proc/net/udp6", "udp", "ipv6"}} {
		if options.Transport != "" && options.Transport != source.transport {
			continue
		}
		items, err := read(source.path, source.transport, source.family, owners)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if options.Port == 0 || item.LocalPort == options.Port {
				records = append(records, item)
			}
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].LocalPort == records[j].LocalPort {
			return records[i].Transport < records[j].Transport
		}
		return records[i].LocalPort < records[j].LocalPort
	})
	return records, nil
}
func read(path, transport, family string, owners map[string]owner) ([]Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, cli.NewError(cli.ExitRuntime, "read %s: %v", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	out := []Record{}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 || fields[3] != "0A" && !(transport == "udp" && fields[3] == "07") {
			continue
		}
		address, port, err := socketAddress(fields[1], family)
		if err != nil {
			continue
		}
		record := Record{Schema: Schema, Status: "ok", Transport: transport, Family: family, LocalAddress: address, LocalPort: port, State: "listen", Visibility: "partial"}
		if value, ok := owners[fields[9]]; ok {
			record.PID = value.pid
			record.Process = value.process
			record.UID = value.uid
			record.Visibility = "complete"
		}
		out = append(out, record)
	}
	return out, scanner.Err()
}

type owner struct {
	pid, uid int
	process  string
}

func owners() map[string]owner {
	result := map[string]owner{}
	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		info, err := os.Stat("/proc/" + entry.Name())
		if err != nil {
			continue
		}
		value := owner{pid: pid, uid: int(info.Sys().(*syscall.Stat_t).Uid)}
		if data, err := os.ReadFile("/proc/" + entry.Name() + "/comm"); err == nil {
			value.process = strings.TrimSpace(string(data))
		}
		fds, _ := os.ReadDir("/proc/" + entry.Name() + "/fd")
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join("/proc", entry.Name(), "fd", fd.Name()))
			if err == nil && strings.HasPrefix(target, "socket:[") {
				result[strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")] = value
			}
		}
	}
	return result
}
func socketAddress(value, family string) (string, int, error) {
	host, portHex, ok := strings.Cut(value, ":")
	if !ok {
		return "", 0, fmt.Errorf("bad address")
	}
	port, err := strconv.ParseInt(portHex, 16, 32)
	if err != nil {
		return "", 0, err
	}
	if family == "ipv4" {
		b := make([]byte, 4)
		for i := 0; i < 4; i++ {
			v, _ := strconv.ParseInt(host[i*2:i*2+2], 16, 8)
			b[3-i] = byte(v)
		}
		return net.IP(b).String(), int(port), nil
	}
	b := make([]byte, 16)
	for i := 0; i < 16; i++ {
		v, _ := strconv.ParseInt(host[i*2:i*2+2], 16, 8)
		b[i] = byte(v)
	}
	return net.IP(b).String(), int(port), nil
}
