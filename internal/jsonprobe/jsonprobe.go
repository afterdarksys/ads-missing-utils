// Package jsonprobe runs bounded, declarative readiness checks. Its check
// language intentionally has no shell-command primitive.
package jsonprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

const Schema = "missing-utils/jsonprobe/v1"
const maxSpecBytes = 8 << 20

type Spec struct {
	Checks []Check `json:"checks"`
}
type Check struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Host        string `json:"host,omitempty"`
	Port        int    `json:"port,omitempty"`
	URL         string `json:"url,omitempty"`
	Path        string `json:"path,omitempty"`
	Exists      *bool  `json:"exists,omitempty"`
	Status      int    `json:"status,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
	Retries     int    `json:"retries,omitempty"`
	Consecutive int    `json:"consecutive_successes,omitempty"`
}
type Record struct {
	Schema     string          `json:"schema"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Status     string          `json:"status"`
	DurationMS int64           `json:"duration_ms"`
	Evidence   map[string]any  `json:"evidence,omitempty"`
	Diagnostic *cli.Diagnostic `json:"diagnostic,omitempty"`
}
type Result struct {
	Schema  string   `json:"schema"`
	Outcome string   `json:"outcome"`
	Checks  []Record `json:"checks"`
}

func Decode(input io.Reader) (Spec, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxSpecBytes+1))
	if err != nil {
		return Spec{}, cli.NewError(cli.ExitRuntime, "read check specification: %v", err)
	}
	if len(data) > maxSpecBytes {
		return Spec{}, cli.NewError(cli.ExitUsage, "check specification exceeds 8 MiB input limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var spec Spec
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, cli.NewError(cli.ExitUsage, "invalid check specification: %v", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return Spec{}, cli.NewError(cli.ExitUsage, "check specification must contain one document")
	}
	if len(spec.Checks) == 0 {
		return Spec{}, cli.NewError(cli.ExitUsage, "check specification requires at least one check")
	}
	for index, check := range spec.Checks {
		if err := validate(check); err != nil {
			return Spec{}, cli.NewError(cli.ExitUsage, "checks[%d]: %v", index, err)
		}
	}
	return spec, nil
}

func Run(ctx context.Context, spec Spec) Result {
	result := Result{Schema: Schema, Outcome: "pass", Checks: make([]Record, 0, len(spec.Checks))}
	for _, check := range spec.Checks {
		record := runCheck(ctx, check)
		result.Checks = append(result.Checks, record)
		if record.Status == "fail" {
			result.Outcome = "fail"
		}
		if record.Status == "error" && result.Outcome == "pass" {
			result.Outcome = "partial"
		}
	}
	return result
}

func validate(check Check) error {
	if check.Name == "" {
		return fmt.Errorf("name is required")
	}
	if check.Retries < 0 || check.Retries > 20 || check.Consecutive < 0 || check.Consecutive > 20 {
		return fmt.Errorf("retries and consecutive_successes must be between 0 and 20")
	}
	if check.Timeout != "" {
		if duration, err := time.ParseDuration(check.Timeout); err != nil || duration <= 0 || duration > 10*time.Minute {
			return fmt.Errorf("timeout must be a duration from 1ns to 10m")
		}
	}
	switch check.Type {
	case "tcp":
		if check.Host == "" || check.Port < 1 || check.Port > 65535 {
			return fmt.Errorf("tcp requires host and port")
		}
	case "http":
		if check.URL == "" {
			return fmt.Errorf("http requires url")
		}
	case "file":
		if check.Path == "" {
			return fmt.Errorf("file requires path")
		}
	case "process":
		if check.Path == "" {
			return fmt.Errorf("process requires an executable name in path")
		}
	default:
		return fmt.Errorf("unsupported type %q", check.Type)
	}
	return nil
}

func runCheck(parent context.Context, check Check) Record {
	started := time.Now()
	timeout := 10 * time.Second
	if check.Timeout != "" {
		timeout, _ = time.ParseDuration(check.Timeout)
	}
	required := check.Consecutive
	if required == 0 {
		required = 1
	}
	record := Record{Schema: Schema, Name: check.Name, Type: check.Type, Status: "fail"}
	successes := 0
	for attempt := 0; attempt <= check.Retries; attempt++ {
		ctx, cancel := context.WithTimeout(parent, timeout)
		evidence, err := probe(ctx, check)
		timedOut := ctx.Err() == context.DeadlineExceeded
		cancel()
		if err == nil {
			successes++
			record.Evidence = evidence
			if successes >= required {
				record.Status = "pass"
				break
			}
		} else {
			successes = 0
			diagnostic := cli.Diagnostic{Code: "probe_failed", Message: err.Error()}
			record.Diagnostic = &diagnostic
			if timedOut {
				diagnostic.Code = "timeout"
				record.Diagnostic = &diagnostic
			}
		}
	}
	record.DurationMS = time.Since(started).Milliseconds()
	return record
}

func probe(ctx context.Context, check Check) (map[string]any, error) {
	switch check.Type {
	case "tcp":
		connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(check.Host, fmt.Sprint(check.Port)))
		if err != nil {
			return nil, err
		}
		defer connection.Close()
		return map[string]any{"address": net.JoinHostPort(check.Host, fmt.Sprint(check.Port))}, nil
	case "http":
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, check.URL, nil)
		if err != nil {
			return nil, err
		}
		response, err := (&http.Client{}).Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		expected := check.Status
		if expected == 0 {
			expected = 200
		}
		if response.StatusCode != expected {
			return map[string]any{"status": response.StatusCode}, fmt.Errorf("expected HTTP status %d, got %d", expected, response.StatusCode)
		}
		return map[string]any{"status": response.StatusCode}, nil
	case "file":
		_, err := os.Stat(check.Path)
		exists := err == nil
		wanted := true
		if check.Exists != nil {
			wanted = *check.Exists
		}
		if exists != wanted {
			return map[string]any{"path": filepath.Clean(check.Path), "exists": exists}, fmt.Errorf("file existence is %t, expected %t", exists, wanted)
		}
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		return map[string]any{"path": filepath.Clean(check.Path), "exists": exists}, nil
	case "process":
		if runtime.GOOS != "linux" {
			return nil, fmt.Errorf("process checks currently require Linux")
		}
		entries, err := os.ReadDir("/proc")
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if _, err := fmt.Sscan(entry.Name(), new(int)); err != nil {
				continue
			}
			data, err := os.ReadFile("/proc/" + entry.Name() + "/comm")
			if err == nil && strings.TrimSpace(string(data)) == check.Path {
				return map[string]any{"process": check.Path, "pid": entry.Name()}, nil
			}
		}
		return nil, fmt.Errorf("process %q not found", check.Path)
	}
	return nil, fmt.Errorf("unsupported check")
}
