// Package osx implements safe, small adapters around selected macOS tools.
package osx

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const Schema = "missing-utils/osx-response/v1"

// Runner makes native command execution testable and keeps command assembly in
// one place. Implementations must not invoke a shell.
type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
	RunInput(context.Context, string, string, ...string) ([]byte, error)
}

type SystemRunner struct{}

func (SystemRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func (SystemRunner) RunInput(ctx context.Context, name, input string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = strings.NewReader(input)
	return command.CombinedOutput()
}

type Request struct {
	Subsystem string
	Action    string
	Args      []string
	Apply     bool
	Duration  time.Duration
}

type Invocation struct {
	Program string   `json:"program"`
	Args    []string `json:"args"`
}

type Result struct {
	Schema      string       `json:"schema"`
	Command     string       `json:"command"`
	Outcome     string       `json:"outcome"`
	Mutated     bool         `json:"mutated"`
	Invocations []Invocation `json:"invocations"`
	Output      string       `json:"output,omitempty"`
}

// Execute runs one supported native operation. State-changing operations need
// Apply, which gives scripts and people the same explicit safety boundary.
func Execute(ctx context.Context, runner Runner, request Request) (Result, error) {
	if runtime.GOOS != "darwin" {
		return Result{}, fmt.Errorf("osx requires macOS (running on %s)", runtime.GOOS)
	}
	program, _, mutate, err := command(request)
	if err != nil {
		return Result{}, err
	}
	if mutate && !request.Apply {
		return Result{}, fmt.Errorf("osx %s %s changes system state; rerun with --apply", request.Subsystem, request.Action)
	}
	result := Result{Schema: Schema, Command: request.Subsystem + " " + request.Action, Outcome: "pass", Mutated: mutate}
	for _, item := range program {
		result.Invocations = append(result.Invocations, Invocation{Program: item.name, Args: item.args})
		var output []byte
		var runErr error
		if item.input == nil {
			output, runErr = runner.Run(ctx, item.name, item.args...)
		} else {
			output, runErr = runner.RunInput(ctx, item.name, *item.input, item.args...)
		}
		if strings.TrimSpace(string(output)) != "" {
			if result.Output != "" {
				result.Output += "\n"
			}
			result.Output += strings.TrimRight(string(output), "\n")
		}
		if runErr != nil {
			return result, fmt.Errorf("run %s: %w", item.name, runErr)
		}
	}
	return result, nil
}

type nativeCommand struct {
	name  string
	args  []string
	input *string
}

func command(r Request) ([]nativeCommand, []string, bool, error) {
	args := r.Args
	one := func(name string, values ...string) ([]nativeCommand, []string, bool, error) {
		return []nativeCommand{{name: name, args: values}}, nil, false, nil
	}
	mutating := func(name string, values ...string) ([]nativeCommand, []string, bool, error) {
		return []nativeCommand{{name: name, args: values}}, nil, true, nil
	}
	switch r.Subsystem + " " + r.Action {
	case "info show":
		return one("sw_vers")
	case "battery status":
		return one("pmset", "-g", "batt")
	case "power settings":
		return one("pmset", "-g", "custom")
	case "network quality":
		return one("networkQuality", "-v")
	case "network interfaces":
		return one("networksetup", "-listallhardwareports")
	case "spotlight search":
		if len(args) != 1 {
			return nil, nil, false, fmt.Errorf("usage: osx spotlight search QUERY")
		}
		return one("mdfind", args[0])
	case "launch open":
		if len(args) != 1 {
			return nil, nil, false, fmt.Errorf("usage: osx launch open PATH_OR_URL")
		}
		return mutating("open", args[0])
	case "dns status":
		return one("scutil", "--dns")
	case "dns flush":
		return []nativeCommand{{name: "dscacheutil", args: []string{"-flushcache"}}, {name: "killall", args: []string{"-HUP", "mDNSResponder"}}}, nil, true, nil
	case "security filevault":
		return one("fdesetup", "status")
	case "security gatekeeper":
		return one("spctl", "--status")
	case "security firewall":
		return one("defaults", "read", "/Library/Preferences/com.apple.alf", "globalstate")
	case "security sip":
		return one("csrutil", "status")
	case "updates list":
		return one("softwareupdate", "--list")
	case "updates install":
		return mutating("softwareupdate", "--install", "--all")
	case "caffeinate start":
		if len(args) != 0 {
			return nil, nil, false, fmt.Errorf("caffeinate start accepts no positional arguments; use --duration")
		}
		values := []string{"-dimsu"}
		if r.Duration > 0 {
			values = append(values, "-t", fmt.Sprintf("%d", int(r.Duration.Seconds())))
		}
		return mutating("caffeinate", values...)
	case "speech say":
		if len(args) != 1 {
			return nil, nil, false, fmt.Errorf("usage: osx speech say TEXT")
		}
		return mutating("say", args[0])
	case "clipboard read":
		if len(args) != 0 {
			return nil, nil, false, fmt.Errorf("usage: osx clipboard read")
		}
		return one("pbpaste")
	case "clipboard write":
		if len(args) != 1 {
			return nil, nil, false, fmt.Errorf("usage: osx clipboard write TEXT")
		}
		return []nativeCommand{{name: "pbcopy", input: &args[0]}}, nil, true, nil
	case "files copy":
		if len(args) != 2 {
			return nil, nil, false, fmt.Errorf("usage: osx files copy SOURCE DESTINATION")
		}
		return mutating("ditto", args[0], args[1])
	case "automation run":
		if len(args) != 1 {
			return nil, nil, false, fmt.Errorf("usage: osx automation run SCRIPT")
		}
		return mutating("osascript", args[0])
	case "defaults read":
		if len(args) < 1 || len(args) > 2 {
			return nil, nil, false, fmt.Errorf("usage: osx defaults read DOMAIN [KEY]")
		}
		return one("defaults", append([]string{"read"}, args...)...)
	case "defaults write":
		if len(args) != 4 {
			return nil, nil, false, fmt.Errorf("usage: osx defaults write DOMAIN KEY {string|bool|int|float} VALUE")
		}
		typeFlag, ok := map[string]string{"string": "-string", "bool": "-bool", "int": "-int", "float": "-float"}[args[2]]
		if !ok {
			return nil, nil, false, fmt.Errorf("defaults write type must be string, bool, int, or float")
		}
		return mutating("defaults", "write", args[0], args[1], typeFlag, args[3])
	case "defaults delete":
		if len(args) < 1 || len(args) > 2 {
			return nil, nil, false, fmt.Errorf("usage: osx defaults delete DOMAIN [KEY]")
		}
		return mutating("defaults", append([]string{"delete"}, args...)...)
	case "volume get":
		return one("osascript", "-e", "output volume of (get volume settings)")
	case "volume set":
		if len(args) != 1 {
			return nil, nil, false, fmt.Errorf("usage: osx volume set PERCENT")
		}
		value, err := strconv.Atoi(args[0])
		if err != nil || value < 0 || value > 100 {
			return nil, nil, false, fmt.Errorf("volume percent must be an integer from 0 through 100")
		}
		return mutating("osascript", "-e", "set volume output volume "+args[0])
	case "volume mute":
		return mutating("osascript", "-e", "set volume output muted true")
	case "volume unmute":
		return mutating("osascript", "-e", "set volume output muted false")
	}
	return nil, nil, false, fmt.Errorf("unknown osx command: %s %s", r.Subsystem, r.Action)
}
