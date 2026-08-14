package main

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
)

type call struct {
	name string
	args []string
}
type fakeRunner struct{ calls []call }

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, call{name, args})
	return []byte("ok\n"), nil
}
func (f *fakeRunner) RunInput(ctx context.Context, name, input string, args ...string) ([]byte, error) {
	return f.Run(ctx, name, args...)
}

func TestDefaultsReadAndWriteSafety(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native command adapter is macOS-only")
	}
	fake := &fakeRunner{}
	var out, err bytes.Buffer
	if got := run([]string{"defaults", "read", "com.apple.finder", "AppleShowAllFiles"}, &out, &err, fake); got != 0 {
		t.Fatalf("read exit = %d: %s", got, err.String())
	}
	if len(fake.calls) != 1 || fake.calls[0].name != "defaults" || strings.Join(fake.calls[0].args, " ") != "read com.apple.finder AppleShowAllFiles" {
		t.Fatalf("unexpected call: %#v", fake.calls)
	}
	out.Reset()
	err.Reset()
	if got := run([]string{"defaults", "write", "com.apple.finder", "AppleShowAllFiles", "bool", "true"}, &out, &err, fake); got == 0 || !strings.Contains(err.String(), "--apply") {
		t.Fatalf("write should require apply: exit=%d err=%q", got, err.String())
	}
	if got := run([]string{"--apply", "defaults", "write", "com.apple.finder", "AppleShowAllFiles", "bool", "true"}, &out, &err, fake); got != 0 {
		t.Fatalf("applied write exit = %d: %s", got, err.String())
	}
}

func TestCaffeinateDuration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native command adapter is macOS-only")
	}
	fake := &fakeRunner{}
	var out, err bytes.Buffer
	if got := run([]string{"caffeinate", "start", "--duration", "2m", "--apply"}, &out, &err, fake); got != 0 {
		t.Fatalf("exit = %d: %s", got, err.String())
	}
	if len(fake.calls) != 1 || strings.Join(fake.calls[0].args, " ") != "-dimsu -t 120" {
		t.Fatalf("unexpected command: %#v", fake.calls)
	}
}
