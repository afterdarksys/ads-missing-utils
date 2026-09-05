// Package runreceipt records local process evidence without inferring remote
// effects or successful business outcomes. It never retries commands.
package runreceipt

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const Schema = "missing-utils/runreceipt/v1"

type Stage struct {
	Index         int    `json:"index"`
	Executable    string `json:"executable"`
	ArgumentCount int    `json:"argument_count"`
	ArgvSHA256    string `json:"argv_sha256"`
	State         string `json:"state"`
	ExitCode      *int   `json:"exit_code"`
}
type Receipt struct {
	Schema          string  `json:"schema"`
	Label           string  `json:"label"`
	Scope           string  `json:"scope"`
	Host            string  `json:"host"`
	Directory       string  `json:"directory"`
	State           string  `json:"state"`
	Reason          string  `json:"reason,omitempty"`
	StartedAt       string  `json:"started_at,omitempty"`
	FinishedAt      string  `json:"finished_at,omitempty"`
	OutcomeVerified bool    `json:"outcome_verified"`
	Stages          []Stage `json:"stages"`
}

func Preview(commands [][]string, label, dir string) (Receipt, error) {
	r := Receipt{Schema: Schema, Label: label, Scope: "local_process_only", Directory: dir, State: "preview", Stages: []Stage{}}
	if len(commands) < 1 || len(commands) > 16 {
		return r, fmt.Errorf("provide between 1 and 16 stages")
	}
	host, err := os.Hostname()
	if err != nil {
		return r, fmt.Errorf("cannot identify local host")
	}
	r.Host = host
	for i, args := range commands {
		if len(args) == 0 || args[0] == "" {
			return r, fmt.Errorf("stage %d has no executable", i+1)
		}
		encoded, err := json.Marshal(args)
		if err != nil {
			return r, err
		}
		r.Stages = append(r.Stages, Stage{Index: i + 1, Executable: args[0], ArgumentCount: len(args) - 1, ArgvSHA256: fmt.Sprintf("%x", sha256.Sum256(encoded)), State: "not_started"})
	}
	return r, nil
}

type lockedWriter struct {
	mu   *sync.Mutex
	dest io.Writer
}

func (w lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dest.Write(p)
}

// Execute persists an intent record before launching. If final persistence fails,
// the original running record remains evidence that completion is unknown.
func Execute(parent context.Context, commands [][]string, label, dir string, stdin io.Reader, stdout, stderr io.Writer, save func(Receipt) error) (Receipt, error) {
	r, err := Preview(commands, label, dir)
	if err != nil {
		return r, err
	}
	r.State = "running"
	r.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := save(r); err != nil {
		return r, err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var mu sync.Mutex
	var pipes [][2]*os.File
	closePipes := func() {
		for _, pair := range pipes {
			_ = pair[0].Close()
			_ = pair[1].Close()
		}
	}
	defer closePipes()
	for i := 1; i < len(commands); i++ {
		reader, writer, err := os.Pipe()
		if err != nil {
			r.State = "incomplete"
			r.Reason = "pipeline_setup_failed"
			r.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
			return r, save(r)
		}
		pipes = append(pipes, [2]*os.File{reader, writer})
	}
	processes := make([]*exec.Cmd, len(commands))
	started := make([]bool, len(commands))
	for i, args := range commands {
		command := exec.CommandContext(ctx, args[0], args[1:]...)
		command.Dir = dir
		command.WaitDelay = 500 * time.Millisecond
		command.Stderr = lockedWriter{&mu, stderr}
		if i == 0 {
			command.Stdin = stdin
		} else {
			command.Stdin = pipes[i-1][0]
		}
		if i == len(commands)-1 {
			command.Stdout = lockedWriter{&mu, stdout}
		} else {
			command.Stdout = pipes[i][1]
		}
		processes[i] = command
	}
	// Start consumers first, then close the parent's copies of all pipe handles.
	for i := len(processes) - 1; i >= 0; i-- {
		if err := processes[i].Start(); err != nil {
			r.State = "incomplete"
			r.Reason = "stage_launch_failed"
			cancel()
			break
		}
		started[i] = true
		r.Stages[i].State = "running"
	}
	closePipes()
	for i, process := range processes {
		if !started[i] {
			continue
		}
		err := process.Wait()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) && exit.ExitCode() >= 0 {
				code = exit.ExitCode()
			} else {
				r.Stages[i].State = "unknown"
				r.State = "incomplete"
				if r.Reason == "" {
					r.Reason = "process_completion_unknown"
				}
				continue
			}
		}
		r.Stages[i].State = "exited"
		r.Stages[i].ExitCode = &code
	}
	if parent.Err() != nil {
		r.State = "incomplete"
		r.Reason = "interrupted"
		if errors.Is(parent.Err(), context.DeadlineExceeded) {
			r.Reason = "timeout"
		}
	}
	if r.State == "running" {
		r.State = "completed"
	}
	r.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return r, save(r)
}
func ExitCode(r Receipt) int {
	if r.Reason == "timeout" {
		return 124
	}
	if r.Reason == "interrupted" {
		return 130
	}
	if r.State != "completed" {
		return 4
	}
	for _, s := range r.Stages {
		if s.ExitCode == nil {
			return 4
		}
		if *s.ExitCode != 0 {
			return 1
		}
	}
	return 0
}
