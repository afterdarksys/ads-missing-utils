package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var v1Binaries map[string]string

func TestMain(m *testing.M) {
	root := repositoryRootForAcceptance()
	binDir, err := os.MkdirTemp("", "missing-utils-v1-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(binDir)
	v1Binaries = map[string]string{}
	for _, name := range []string{"jwalk", "envsub", "hashsum"} {
		binary := filepath.Join(binDir, name)
		cmd := exec.Command("go", "build", "-o", binary, "./cmd/"+name)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			panic("build " + name + ": " + err.Error() + "\n" + string(output))
		}
		v1Binaries[name] = binary
	}
	os.Exit(m.Run())
}

func TestV1SharedStructuredOutputAndUsage(t *testing.T) {
	root := t.TempDir()
	writeAcceptanceFile(t, filepath.Join(root, "record.txt"), "record")
	stdout, stderr, code := runV1(t, "jwalk", nil, "--format", "json", root)
	if code != 0 || stderr != "" {
		t.Fatalf("jwalk code=%d stderr=%q", code, stderr)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(stdout), &records); err != nil || len(records) == 0 {
		t.Fatalf("jwalk stdout is not one JSON document: %q; err=%v", stdout, err)
	}
	_, _, code = runV1(t, "jwalk", nil, "--format", "invalid", root)
	if code != 2 {
		t.Fatalf("invalid jwalk format exit code=%d, want 2", code)
	}
}

func TestV1JwalkFiltersMetadataAndSchema(t *testing.T) {
	root := t.TempDir()
	writeAcceptanceFile(t, filepath.Join(root, "a-match.log"), "a")
	writeAcceptanceFile(t, filepath.Join(root, "b-match.log"), "bb")
	writeAcceptanceFile(t, filepath.Join(root, "skip.txt"), "skip")
	stdout, stderr, code := runV1(t, "jwalk", nil, "--format", "json", "--type", "file", "--include", `\.log$`, "--max-size", "2", root)
	if code != 0 || stderr != "" {
		t.Fatalf("jwalk code=%d stderr=%q", code, stderr)
	}
	var records []struct {
		Schema string `json:"schema"`
		Status string `json:"status"`
		Path   string `json:"path"`
		Type   string `json:"type"`
	}
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0].Path > records[1].Path {
		t.Fatalf("filtered records=%#v", records)
	}
	for _, record := range records {
		if record.Schema != "missing-utils/jwalk/v1" || record.Status != "ok" || record.Type != "file" {
			t.Fatalf("invalid jwalk record=%#v", record)
		}
	}
}

func TestV1EnvsubPrecedenceValidationAndNonWritingModes(t *testing.T) {
	dir := t.TempDir()
	template := filepath.Join(dir, "app.tmpl")
	output := filepath.Join(dir, "app.conf")
	envFile := filepath.Join(dir, ".env")
	schema := filepath.Join(dir, "schema.yaml")
	writeAcceptanceFile(t, template, "${VALUE}|${PORT}|${TOKEN}")
	writeAcceptanceFile(t, output, "old")
	writeAcceptanceFile(t, envFile, "VALUE=file\nPORT=8080\nTOKEN=top-secret\n")
	writeAcceptanceFile(t, schema, "PORT:\n  type: integer\nTOKEN:\n  type: string\n  secret: true\n")
	stdout, stderr, code := runV1(t, "envsub", nil, "--input", template, "--output", output, "--env-file", envFile, "--schema", schema, "--set", "VALUE=set")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"schema":"missing-utils/envsub/v1"`) {
		t.Fatalf("envsub code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	contents, err := os.ReadFile(output)
	if err != nil || string(contents) != "set|8080|top-secret" {
		t.Fatalf("rendered output=%q err=%v", contents, err)
	}
	writeAcceptanceFile(t, output, "unchanged")
	_, _, code = runV1(t, "envsub", nil, "--input", template, "--output", output, "--env-file", envFile, "--schema", schema, "--check")
	contents, _ = os.ReadFile(output)
	if code != 0 || string(contents) != "unchanged" {
		t.Fatalf("check code=%d output=%q", code, contents)
	}
	_, stderr, code = runV1(t, "envsub", nil, "--input", template, "--schema", schema, "--set", "PORT=bad", "--set", "TOKEN=top-secret")
	if code != 2 || strings.Contains(stderr, "top-secret") {
		t.Fatalf("validation code=%d stderr=%q", code, stderr)
	}
}

func TestV1HashsumPipelineAndVerification(t *testing.T) {
	root := t.TempDir()
	writeAcceptanceFile(t, filepath.Join(root, "a.txt"), "alpha")
	writeAcceptanceFile(t, filepath.Join(root, "b.txt"), "bravo")
	manifest := filepath.Join(root, "manifest.json")
	jwalk, stderr, code := runV1(t, "jwalk", nil, "--type", "file", "--format", "ndjson", root)
	if code != 0 || stderr != "" {
		t.Fatalf("jwalk pipeline code=%d stderr=%q", code, stderr)
	}
	stdout, stderr, code := runV1(t, "hashsum", strings.NewReader(jwalk), "create", "--from-jwalk", "--root", root, "--workers", "2", "--progress", "--output", manifest)
	if code != 0 || stdout != "" || !strings.Contains(stderr, "hashed") {
		t.Fatalf("hashsum create code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	data, err := os.ReadFile(manifest)
	if err != nil || !bytes.Contains(data, []byte(`"sha256"`)) || !bytes.Contains(data, []byte(`"blake3"`)) {
		t.Fatalf("manifest=%q err=%v", data, err)
	}
	stdout, stderr, code = runV1(t, "hashsum", nil, "verify", "--root", root, "--unexpected=false", manifest)
	if code != 0 || stderr != "" {
		t.Fatalf("hashsum verify code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var verify []map[string]any
	if err := json.Unmarshal([]byte(stdout), &verify); err != nil || len(verify) != 2 {
		t.Fatalf("verify output=%q err=%v", stdout, err)
	}
}

func TestV1SchemasValidateRealCommandOutput(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	writeAcceptanceFile(t, file, "content")

	stdout, stderr, code := runV1(t, "jwalk", nil, "--format", "json", "--type", "file", root)
	if code != 0 || stderr != "" {
		t.Fatalf("jwalk code=%d stderr=%q", code, stderr)
	}
	var records []json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &records); err != nil || len(records) != 1 {
		t.Fatalf("jwalk output=%q err=%v", stdout, err)
	}
	validateJSONSchema(t, "jwalk-v1.json", records[0])

	template := filepath.Join(root, "template")
	writeAcceptanceFile(t, template, "${VALUE}")
	stdout, stderr, code = runV1(t, "envsub", nil, "--input", template, "--set", "VALUE=ok")
	if code != 0 || stderr != "" {
		t.Fatalf("envsub code=%d stderr=%q", code, stderr)
	}
	validateJSONSchema(t, "envsub-v1.json", json.RawMessage(stdout))

	stdout, stderr, code = runV1(t, "hashsum", nil, "create", "--root", root, "file.txt")
	if code != 0 || stderr != "" {
		t.Fatalf("hashsum code=%d stderr=%q", code, stderr)
	}
	validateJSONSchema(t, "hashsum-v1.json", json.RawMessage(stdout))
	validateJSONSchema(t, "response-v1.json", json.RawMessage(`{"schema":"missing-utils/response/v1","command":"jwalk","outcome":"pass","data":[]}`))
}

func validateJSONSchema(t *testing.T, name string, document json.RawMessage) {
	t.Helper()
	root := repositoryRootForAcceptance()
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	schema, err := compiler.Compile(filepath.Join(root, "schemas", name))
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(document))
	if err != nil {
		t.Fatalf("decode %s instance: %v", name, err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Fatalf("validate %s: %v\ndocument: %s", name, err, document)
	}
}

func runV1(t *testing.T, name string, input *strings.Reader, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(v1Binaries[name], args...)
	if input != nil {
		cmd.Stdin = input
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exit.ExitCode()
	}
	t.Fatalf("run %s: %v", name, err)
	return "", "", 0
}

func writeAcceptanceFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func repositoryRootForAcceptance() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(wd)
}
