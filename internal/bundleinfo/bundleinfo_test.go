package bundleinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectApp(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Demo.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0"?><plist><dict><key>CFBundleIdentifier</key><string>test.demo</string><key>CFBundleExecutable</key><string>Demo</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", "MacOS", "Demo"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "app" || report.BundleIdentifier != "test.demo" || report.Executable != "Demo" || !report.ExecutablePresent || report.FileCount != 2 {
		t.Fatalf("report %#v", report)
	}
}
