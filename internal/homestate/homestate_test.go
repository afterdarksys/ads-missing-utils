package homestate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIndexCompareNewModifiedRemovedAndMoved(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(t.TempDir(), "state.json")
	writeTestFile(t, filepath.Join(root, "same"), "same")
	writeTestFile(t, filepath.Join(root, "modify"), "old")
	writeTestFile(t, filepath.Join(root, "remove"), "gone")
	writeTestFile(t, filepath.Join(root, "old", "move"), "moving")
	if _, err := Index(root, state); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "modify"), "new")
	if err := os.Remove(filepath.Join(root, "remove")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "old", "move"), filepath.Join(root, "moved")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "added"), "added")
	db, err := Load(state)
	if err != nil {
		t.Fatal(err)
	}
	changes, _, err := Check(db, state)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, change := range changes {
		got[change.Path] = change.Status
	}
	for path, want := range map[string]string{"modify": "modified", "remove": "removed", "added": "added", "moved": "moved"} {
		if got[path] != want {
			t.Fatalf("%s = %q, changes %#v", path, got[path], changes)
		}
	}
}

func TestIndexExcludesStateAndUsesRelativePaths(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, ".state.json")
	writeTestFile(t, filepath.Join(root, "dir", "file"), "x")
	db, err := Index(root, state)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := db.Entries[".state.json"]; ok {
		t.Fatal("state indexed itself")
	}
	if _, ok := db.Entries["dir/file"]; !ok {
		t.Fatalf("entries %#v", db.Entries)
	}
}

func TestStateInsideRootDoesNotCreateImmediateChanges(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, ".local", "state", "missing-utils", "homestate.json")
	writeTestFile(t, filepath.Join(root, "file"), "x")
	db, err := Index(root, state)
	if err != nil {
		t.Fatal(err)
	}
	changes, diagnostics, err := Check(db, state)
	if err != nil || len(diagnostics) != 0 || len(changes) != 0 {
		t.Fatalf("changes=%#v diagnostics=%#v err=%v", changes, diagnostics, err)
	}
}

func TestBrokenFixAndRestoreNames(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(t.TempDir(), "state.json")
	original := filepath.Join(root, "bad:name", "quote'file")
	writeTestFile(t, original, "x")
	broken, err := BrokenNames(root)
	if err != nil || len(broken) != 2 {
		t.Fatalf("broken=%#v err=%v", broken, err)
	}
	renamed, diagnostics, err := FixNames(root, state)
	if err != nil || len(diagnostics) != 0 || len(renamed) != 2 {
		t.Fatalf("renamed=%#v diagnostics=%#v err=%v", renamed, diagnostics, err)
	}
	if _, err := os.Stat(filepath.Join(root, "bad_name", "quote_file")); err != nil {
		t.Fatal(err)
	}
	restored, diagnostics, err := RestoreNames(root, state)
	if err != nil || len(diagnostics) != 0 || len(restored) != 2 {
		t.Fatalf("restored=%#v diagnostics=%#v err=%v", restored, diagnostics, err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatal(err)
	}
}

func TestFixNamesDoesNotOverwriteCollision(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(t.TempDir(), "state.json")
	writeTestFile(t, filepath.Join(root, "bad:name"), "one")
	writeTestFile(t, filepath.Join(root, "bad_name"), "two")
	renamed, diagnostics, err := FixNames(root, state)
	if err != nil || len(renamed) != 1 || len(diagnostics) != 0 {
		t.Fatalf("renamed=%#v diagnostics=%#v err=%v", renamed, diagnostics, err)
	}
	if renamed[0].To == "bad_name" {
		t.Fatal("collision was not disambiguated")
	}
}
