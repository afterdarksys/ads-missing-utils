// Package pwatch provides passive, bounded Linux process observations.
package pwatch

import (
	"bufio"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const Schema = "missing-utils/pwatch/v1"

type Record struct {
	Schema     string          `json:"schema"`
	Status     string          `json:"status"`
	PID        int             `json:"pid"`
	Timestamp  time.Time       `json:"timestamp"`
	State      string          `json:"state,omitempty"`
	RSSBytes   int64           `json:"rss_bytes,omitempty"`
	Threads    int             `json:"threads,omitempty"`
	Command    string          `json:"command,omitempty"`
	Diagnostic *cli.Diagnostic `json:"diagnostic,omitempty"`
}

func Sample(pid int) (Record, error) {
	if runtime.GOOS != "linux" {
		return Record{}, cli.NewError(cli.ExitRuntime, "pwatch is currently implemented for Linux")
	}
	record := Record{Schema: Schema, Status: "ok", PID: pid, Timestamp: time.Now().UTC()}
	status, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		if os.IsNotExist(err) {
			record.Status = "exited"
			return record, nil
		}
		return Record{}, cli.NewError(cli.ExitRuntime, "read process: %v", err)
	}
	defer status.Close()
	scanner := bufio.NewScanner(status)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch key {
		case "State":
			record.State = strings.Fields(value)[0]
		case "VmRSS":
			fields := strings.Fields(value)
			if len(fields) > 0 {
				kb, _ := strconv.ParseInt(fields[0], 10, 64)
				record.RSSBytes = kb * 1024
			}
		case "Threads":
			record.Threads, _ = strconv.Atoi(value)
		case "Name":
			record.Command = value
		}
	}
	return record, scanner.Err()
}
