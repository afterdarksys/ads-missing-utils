// Package eventwhy groups Docker event evidence without guessing root causes.
package eventwhy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/collectout"
	"github.com/afterdarksys/ads-missing-utils/internal/contextsnap"
	"io"
	"sort"
	"strings"
	"time"
)

const Schema = "missing-utils/eventwhy/v1"

type Group struct {
	Resource string    `json:"resource"`
	Action   string    `json:"action"`
	ExitCode string    `json:"exit_code,omitempty"`
	Count    int       `json:"count"`
	First    time.Time `json:"first"`
	Last     time.Time `json:"last"`
}
type Result struct {
	Schema           string    `json:"schema"`
	Source           string    `json:"source"`
	Context          string    `json:"context"`
	CollectedAt      time.Time `json:"collected_at"`
	CollectorVersion string    `json:"collector_version"`
	Outcome          string    `json:"outcome"`
	Events           int       `json:"events"`
	Ignored          int       `json:"ignored"`
	Groups           []Group   `json:"groups"`
	Diagnostics      []string  `json:"diagnostics"`
}

func Normalize(reader io.Reader, source, dockerContext string, maxEvents int) Result {
	r := Result{Schema: Schema, Source: source, Context: dockerContext, CollectedAt: time.Now().UTC(), CollectorVersion: cli.Version, Outcome: "pass", Groups: []Group{}, Diagnostics: []string{}}
	scanner := bufio.NewScanner(io.LimitReader(reader, collectout.Limit+1))
	scanner.Buffer(make([]byte, 4096), 256<<10)
	positions := map[string]int{}
	lines, bytesRead := 0, 0
	for scanner.Scan() {
		bytesRead += len(scanner.Bytes()) + 1
		if bytesRead > collectout.Limit {
			r.Diagnostics = append(r.Diagnostics, "input byte limit reached; collection is incomplete")
			break
		}
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		lines++
		if lines > maxEvents {
			r.Diagnostics = append(r.Diagnostics, "event limit reached; collection is incomplete")
			break
		}
		var event struct {
			Type   string `json:"Type"`
			Action string `json:"Action"`
			Time   int64  `json:"time"`
			Actor  struct {
				ID         string            `json:"ID"`
				Attributes map[string]string `json:"Attributes"`
			} `json:"Actor"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			r.Diagnostics = append(r.Diagnostics, fmt.Sprintf("event %d is malformed", lines))
			continue
		}
		if !contextsnap.Safe(event.Type) {
			r.Diagnostics = append(r.Diagnostics, fmt.Sprintf("event %d has no usable type", lines))
			continue
		}
		if event.Type != "container" {
			r.Ignored++
			continue
		}
		if !contextsnap.Safe(event.Actor.ID) || !contextsnap.Safe(event.Action) || event.Time <= 0 {
			r.Diagnostics = append(r.Diagnostics, fmt.Sprintf("event %d lacks identity, action, or time", lines))
			continue
		}
		exit := event.Actor.Attributes["exitCode"]
		if len(exit) > 8 || strings.IndexFunc(exit, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
			exit = ""
		}
		key := event.Actor.ID + "\x00" + event.Action + "\x00" + exit
		at := time.Unix(event.Time, 0).UTC()
		r.Events++
		if i, ok := positions[key]; ok {
			g := &r.Groups[i]
			g.Count++
			if at.Before(g.First) {
				g.First = at
			}
			if at.After(g.Last) {
				g.Last = at
			}
		} else {
			positions[key] = len(r.Groups)
			r.Groups = append(r.Groups, Group{Resource: event.Actor.ID, Action: event.Action, ExitCode: exit, Count: 1, First: at, Last: at})
		}
	}
	if scanner.Err() != nil {
		r.Diagnostics = append(r.Diagnostics, "event input failed or a record exceeded 256 KiB")
	}
	if len(r.Diagnostics) > 0 {
		r.Outcome = "partial"
	}
	sort.Slice(r.Groups, func(i, j int) bool {
		a, b := r.Groups[i], r.Groups[j]
		if a.Resource != b.Resource {
			return a.Resource < b.Resource
		}
		if a.Action != b.Action {
			return a.Action < b.Action
		}
		return a.ExitCode < b.ExitCode
	})
	return r
}
func Collect(ctx context.Context, dockerContext string, window time.Duration, maxEvents int, run collectout.Runner) Result {
	end := time.Now().UTC()
	start := end.Add(-window)
	data, err := run(ctx, "docker", "--context", dockerContext, "events", "--since", start.Format(time.RFC3339Nano), "--until", end.Format(time.RFC3339Nano), "--filter", "type=container", "--format", "{{json .}}")
	if err != nil {
		r := Normalize(strings.NewReader(""), "cli", dockerContext, maxEvents)
		r.Outcome = "partial"
		r.Diagnostics = append(r.Diagnostics, "Docker event collection failed or timed out")
		return r
	}
	r := Normalize(strings.NewReader(string(data)), "cli", dockerContext, maxEvents)
	// Docker returns a limited event history, so absence must not imply complete history.
	r.Outcome = "partial"
	r.Diagnostics = append(r.Diagnostics, "Docker retains limited event history; completeness of the requested interval is unknown")
	return r
}
