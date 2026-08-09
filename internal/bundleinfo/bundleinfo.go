package bundleinfo

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/pll"
)

type Report struct {
	Schema               string   `json:"schema"`
	Path                 string   `json:"path"`
	Kind                 string   `json:"kind"`
	BundleIdentifier     string   `json:"bundle_identifier,omitempty"`
	Executable           string   `json:"executable,omitempty"`
	ExecutablePresent    bool     `json:"executable_present,omitempty"`
	InfoPlistPresent     bool     `json:"info_plist_present,omitempty"`
	CodeSignaturePresent bool     `json:"code_signature_present,omitempty"`
	FileCount            int      `json:"file_count"`
	DirectoryCount       int      `json:"directory_count"`
	PackageMetadata      string   `json:"package_metadata,omitempty"`
	Findings             []string `json:"findings,omitempty"`
}

func Inspect(path string) (Report, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Report{}, err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return Report{}, err
	}
	report := Report{Schema: "missing-utils/bundleinfo/v1", Path: absolute, Findings: []string{}}
	lower := strings.ToLower(absolute)
	switch {
	case info.IsDir() && strings.HasSuffix(lower, ".app"):
		report.Kind = "app"
		err = inspectApp(&report)
	case strings.HasSuffix(lower, ".pkg"):
		report.Kind = "pkg"
		err = inspectPkg(&report)
	default:
		return Report{}, fmt.Errorf("path must be an .app directory or .pkg path")
	}
	return report, err
}
func inspectApp(report *Report) error {
	plistPath := filepath.Join(report.Path, "Contents", "Info.plist")
	data, err := os.ReadFile(plistPath)
	if err == nil {
		report.InfoPlistPresent = true
		values, _, parseErr := pll.Parse(data)
		if parseErr != nil {
			report.Findings = append(report.Findings, "Info.plist: "+parseErr.Error())
		} else {
			report.BundleIdentifier, _ = values["CFBundleIdentifier"].(string)
			report.Executable, _ = values["CFBundleExecutable"].(string)
		}
	} else {
		report.Findings = append(report.Findings, "Contents/Info.plist is missing or unreadable")
	}
	if report.Executable != "" {
		info, statErr := os.Stat(filepath.Join(report.Path, "Contents", "MacOS", report.Executable))
		report.ExecutablePresent = statErr == nil && !info.IsDir()
		if !report.ExecutablePresent {
			report.Findings = append(report.Findings, "declared executable is missing")
		}
	}
	if info, err := os.Stat(filepath.Join(report.Path, "Contents", "_CodeSignature", "CodeResources")); err == nil && !info.IsDir() {
		report.CodeSignaturePresent = true
	}
	return filepath.WalkDir(report.Path, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == report.Path {
			return nil
		}
		if entry.IsDir() {
			report.DirectoryCount++
		} else {
			report.FileCount++
		}
		return nil
	})
}
func inspectPkg(report *Report) error {
	info, err := os.Lstat(report.Path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		err = filepath.WalkDir(report.Path, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == report.Path {
				return nil
			}
			if entry.IsDir() {
				report.DirectoryCount++
			} else {
				report.FileCount++
			}
			return nil
		})
	} else {
		report.FileCount = 1
	}
	if err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		if binary, lookErr := exec.LookPath("pkgutil"); lookErr == nil {
			output, cmdErr := exec.Command(binary, "--check-signature", report.Path).CombinedOutput()
			report.PackageMetadata = string(output)
			if cmdErr != nil {
				report.Findings = append(report.Findings, "pkgutil signature metadata unavailable")
			}
		}
	} else {
		report.Findings = append(report.Findings, "package metadata enrichment requires macOS pkgutil")
	}
	return nil
}
