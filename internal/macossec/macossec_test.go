package macossec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func put(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}
func TestCLTDetectsHiddenExecutableHashTwin(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "visible"), "payload", 0o755)
	put(t, filepath.Join(root, ".hidden"), "payload", 0o755)
	report, err := ScanCloneExecutables(root)
	if err != nil || len(report.Candidates) != 1 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}
func TestDTPRejectsTamperAndReportsChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "daemon")
	put(t, path, "one", 0o755)
	key := []byte("0123456789abcdef")
	baseline, err := CreateDaemonBaseline(root, key)
	if err != nil {
		t.Fatal(err)
	}
	put(t, path, "two", 0o755)
	changes, err := CheckDaemonBaseline(baseline, key)
	if err != nil || len(changes) != 1 || changes[0].Status != "modified" {
		t.Fatalf("changes=%#v err=%v", changes, err)
	}
	baseline.Signature = "bad"
	if _, err := CheckDaemonBaseline(baseline, key); err == nil {
		t.Fatal("tamper accepted")
	}
}
func TestDTPRejectsInvalidVerificationKeysAndSignatures(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "daemon"), "one", 0o755)
	key := []byte("0123456789abcdef")
	baseline, err := CreateDaemonBaseline(root, key)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		key       []byte
		signature string
	}{
		{name: "wrong key", key: []byte("fedcba9876543210"), signature: baseline.Signature},
		{name: "empty key", key: nil, signature: baseline.Signature},
		{name: "truncated signature", key: key, signature: baseline.Signature[:12]},
		{name: "garbage signature", key: key, signature: "not-a-signature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := baseline
			candidate.Signature = tt.signature
			if _, err := CheckDaemonBaseline(candidate, tt.key); err == nil {
				t.Fatal("invalid baseline verification accepted")
			}
		})
	}

	if _, err := CreateDaemonBaseline(root, nil); err == nil {
		t.Fatal("baseline creation accepted an empty key")
	}
}
func TestDTPRecordsUnreadableExecutableAndFailsClosed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "unreadable-daemon")
	put(t, path, "secret", 0o111)

	key := []byte("0123456789abcdef")
	baseline, err := CreateDaemonBaseline(root, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.ScanErrors) == 0 {
		t.Skip("current user can read mode-0111 files")
	}
	if baseline.ScanErrors[0].Path != "unreadable-daemon" {
		t.Fatalf("scan errors = %#v", baseline.ScanErrors)
	}
	if _, err := CheckDaemonBaseline(baseline, key); err == nil || !strings.Contains(err.Error(), "unreadable-daemon") {
		t.Fatalf("check error = %v, want unreadable path", err)
	}
}
func TestTrackerReportsVanishedProcess(t *testing.T) {
	tracker := NewTracker()
	now := time.Now()
	tracker.Observe(now, []Process{{42, "/missing/transient"}})
	ended := tracker.Observe(now.Add(50*time.Millisecond), nil)
	if len(ended) != 1 || !ended[0].ExecutableDeleted {
		t.Fatalf("ended %#v", ended)
	}
}
func TestSecretDetectionDoesNotReturnSecret(t *testing.T) {
	secret := "AKIAABCDEFGHIJKLMNOP"
	matches := DetectSecrets(secret)
	if len(matches) != 1 || matches[0].Fingerprint == secret {
		t.Fatalf("matches %#v", matches)
	}
}
func TestOrdinaryProseIsNotASeedPhrase(t *testing.T) {
	if matches := DetectSecrets("this ordinary sentence has many words but it is not labeled as recovery material at all"); len(matches) != 0 {
		t.Fatalf("false positive %#v", matches)
	}
}
func TestIntentGraph(t *testing.T) {
	graph := MapAppleScriptIntent(`do shell script "curl https://example.com"`)
	if len(graph.Edges) != 2 {
		t.Fatalf("graph %#v", graph)
	}
}
func TestWebKitScan(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "View.swift"), `prefs.setValue(true, forKey: "allowUniversalAccessFromFileURLs")`, 0o644)
	report, err := ScanWebKit(root)
	if err != nil || len(report.Findings) != 2 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}
func TestParseAccessibilityGrants(t *testing.T) {
	grants, err := parseAccessibilityGrants("com.example.tool|2|123\n")
	if err != nil || len(grants) != 1 || grants[0].Client != "com.example.tool" || grants[0].Authorization != 2 {
		t.Fatalf("grants=%#v err=%v", grants, err)
	}
}
