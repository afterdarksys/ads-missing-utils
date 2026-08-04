package envsub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

func TestRunPrecedenceDefaultsAndAtomicOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.tmpl")
	output := filepath.Join(dir, "output")
	if err := os.WriteFile(input, []byte("${A}|${B:-fallback}|${C}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte("A=file\nC=filec\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	schema := filepath.Join(dir, "schema.yaml")
	if err := os.WriteFile(schema, []byte("C:\n  type: string\n  default: defaultc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A", "environment")
	result, err := Run(Options{Input: input, Output: output, EnvFiles: []string{env}, Sets: []string{"A=set"}, SchemaPath: schema})
	if err != nil {
		t.Fatal(err)
	}
	if result.Rendered != "set|fallback|filec" {
		t.Fatalf("rendered = %q", result.Rendered)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "set|fallback|filec" {
		t.Fatalf("output = %q", data)
	}
	if info, _ := os.Stat(output); info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestRunValidationSecretAndNonWritingModes(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in")
	output := filepath.Join(dir, "out")
	schema := filepath.Join(dir, "schema.yaml")
	if err := os.WriteFile(input, []byte("${PORT}|${TOKEN}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schema, []byte("PORT:\n  type: integer\nTOKEN:\n  type: string\n  secret: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Run(Options{Input: input, Output: output, SchemaPath: schema, Sets: []string{"PORT=1.2", "TOKEN=very-secret"}})
	if cli.ExitCode(err) != cli.ExitUsage || strings.Contains(err.Error(), "very-secret") {
		t.Fatalf("error = %v", err)
	}
	data, _ := os.ReadFile(output)
	if string(data) != "unchanged" {
		t.Fatalf("output changed after failure")
	}
	result, err := Run(Options{Input: input, Output: output, SchemaPath: schema, Sets: []string{"PORT=12", "TOKEN=very-secret"}, Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Resolutions[1].Value != "***" {
		t.Fatalf("secret resolution = %#v", result.Resolutions)
	}
	data, _ = os.ReadFile(output)
	if string(data) != "unchanged" {
		t.Fatalf("output changed in explain mode")
	}
}

func TestInvalidTemplateAndEnvFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in")
	if err := os.WriteFile(input, []byte("${A:?bad}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Input: input}); cli.ExitCode(err) != cli.ExitUsage {
		t.Fatalf("unsupported placeholder error = %v", err)
	}
	bad := filepath.Join(dir, ".env")
	if err := os.WriteFile(bad, []byte("not-an-assignment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Input: input, EnvFiles: []string{bad}}); cli.ExitCode(err) != cli.ExitUsage {
		t.Fatalf("malformed env error = %v", err)
	}
}
