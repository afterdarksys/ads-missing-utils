package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

func TestRunTransformsPositionalText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"--prefix", "--seed", "1", "hello world"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != cli.ExitOK {
		t.Fatalf("exit = %d, stderr = %q", exitCode, stderr.String())
	}
	if got := stdout.String(); got != "meow: hello world\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunTransformsStandardInputAsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"--format", "json", "--seed", "1"}, strings.NewReader("from stdin"), &stdout, &stderr)
	if exitCode != cli.ExitOK {
		t.Fatalf("exit = %d, stderr = %q", exitCode, stderr.String())
	}
	var response struct {
		Schema string `json:"schema"`
		Data   struct {
			Output string `json:"output"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Schema != "missing-utils/meow/v1" || response.Data.Output != "from stdin" {
		t.Fatalf("response = %#v", response)
	}
}

func TestRunRejectsInvalidFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exitCode := run([]string{"--format", "xml"}, strings.NewReader("text"), &stdout, &stderr); exitCode != cli.ExitUsage {
		t.Fatalf("exit = %d, stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
