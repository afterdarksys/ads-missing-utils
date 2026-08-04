package hashsum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

func TestCreateFromJwalkAndVerify(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader(`{"path":"a.txt","status":"ok","type":"file"}` + "\n" + `{"path":"b.txt","status":"ok","type":"file"}` + "\n")
	manifest, err := Create(CreateOptions{Root: root, FromJwalk: true, Workers: 2}, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 2 || manifest.Files[0].Path != "a.txt" || manifest.Files[0].Digests["sha256"] == "" || manifest.Files[0].Digests["blake3"] == "" {
		t.Fatalf("manifest = %#v", manifest)
	}
	path := filepath.Join(root, "manifest.json")
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	records, err := Verify(VerifyOptions{Root: root, ManifestPath: path, IncludeUnexpected: false})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		if record.Status != "ok" {
			t.Fatalf("verify = %#v", records)
		}
	}
}

func TestVerifyFailuresAndPathSafety(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "changed"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{Schema: Schema, Algorithms: []string{"sha256", "blake3"}, Root: root, Files: []File{{Path: "changed", Digests: map[string]string{"sha256": "bad"}}, {Path: "missing", Digests: map[string]string{"sha256": "bad"}}}}
	path := filepath.Join(root, "manifest.json")
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	records, err := Verify(VerifyOptions{Root: root, ManifestPath: path})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, record := range records {
		got[record.Path] = record.Status
	}
	if got["changed"] != "changed" || got["missing"] != "missing" {
		t.Fatalf("records = %#v", records)
	}
	manifest.Files = []File{{Path: "../outside", Digests: map[string]string{"sha256": "bad"}}}
	data, _ = json.Marshal(manifest)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(VerifyOptions{Root: root, ManifestPath: path}); cli.ExitCode(err) != cli.ExitUsage {
		t.Fatalf("escape error = %v", err)
	}
	manifest.Files = []File{{Path: "same", Digests: map[string]string{"sha256": "bad"}}, {Path: "same", Digests: map[string]string{"sha256": "bad"}}}
	data, _ = json.Marshal(manifest)
	_ = os.WriteFile(path, data, 0o600)
	if _, err := Verify(VerifyOptions{Root: root, ManifestPath: path}); cli.ExitCode(err) != cli.ExitUsage {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestCreateRejectsMalformedJwalk(t *testing.T) {
	_, err := Create(CreateOptions{Root: t.TempDir(), FromJwalk: true, Workers: 1}, strings.NewReader(`{"path":""}`+"\n"))
	if cli.ExitCode(err) != cli.ExitUsage {
		t.Fatalf("error = %v", err)
	}
}
