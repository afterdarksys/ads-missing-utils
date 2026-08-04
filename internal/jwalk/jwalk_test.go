package jwalk

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

func TestWalkFiltersAndDeterministicMetadata(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string, age time.Duration) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		when := time.Now().Add(-age)
		if err := os.Chtimes(path, when, when); err != nil {
			t.Fatal(err)
		}
	}
	write("z.log", "123456", 2*time.Hour)
	write("a.log", "12", 2*time.Hour)
	write("nested/old.log", "123456", 48*time.Hour)
	write("nested/skip.txt", "123456", 2*time.Hour)
	min := int64(4)
	old := 3 * time.Hour
	include := regexp.MustCompile(`\.log$`)
	records, err := Walk(Options{Roots: []string{root}, Types: map[string]bool{"file": true}, Include: include, MinSize: &min, NewerThan: &old})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records: %#v", len(records), records)
	}
	if got := records[0]; got.RelativePath == nil || *got.RelativePath != "z.log" || got.Schema != Schema || got.Mode == nil || got.Size == nil || *got.Size != 6 {
		t.Fatalf("unexpected record: %#v", got)
	}
}

func TestWalkMissingRootAndSymlink(t *testing.T) {
	if _, err := Walk(Options{Roots: []string{filepath.Join(t.TempDir(), "missing")}}); cli.ExitCode(err) != cli.ExitUsage {
		t.Fatalf("missing root error = %v, code=%d", err, cli.ExitCode(err))
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".", filepath.Join(root, "cycle")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	records, err := Walk(Options{Roots: []string{root}})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		if record.Path == filepath.Join(root, "cycle") {
			t.Fatalf("symlink was emitted without follow flag")
		}
	}
}

func TestWalkFollowsDirectorySymlinkWithoutLooping(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target", "inside.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	records, err := Walk(Options{Roots: []string{root}, Types: map[string]bool{"file": true}, FollowSymlinks: true})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, record := range records {
		if strings.HasSuffix(record.Path, "inside.txt") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("followed file count = %d, records=%#v", count, records)
	}
}
