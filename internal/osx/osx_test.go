package osx

import (
	"strings"
	"testing"
	"time"
)

func TestCommandBuildsNativeArguments(t *testing.T) {
	commands, _, mutates, err := command(Request{Subsystem: "caffeinate", Action: "start", Duration: 2 * time.Minute})
	if err != nil || !mutates || len(commands) != 1 || commands[0].name != "caffeinate" || strings.Join(commands[0].args, " ") != "-dimsu -t 120" {
		t.Fatalf("unexpected caffeinate command: %#v mutate=%t err=%v", commands, mutates, err)
	}
	commands, _, mutates, err = command(Request{Subsystem: "clipboard", Action: "write", Args: []string{"hello"}})
	if err != nil || !mutates || len(commands) != 1 || commands[0].name != "pbcopy" || commands[0].input == nil || *commands[0].input != "hello" {
		t.Fatalf("unexpected pbcopy command: %#v mutate=%t err=%v", commands, mutates, err)
	}
}

func TestCommandRejectsUnsafeVolumeValue(t *testing.T) {
	_, _, _, err := command(Request{Subsystem: "volume", Action: "set", Args: []string{"20; tell app \"Finder\""}})
	if err == nil || !strings.Contains(err.Error(), "integer") {
		t.Fatalf("expected validation error, got %v", err)
	}
}
