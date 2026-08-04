package hashsum

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/zeebo/blake3"
)

const Schema = "missing-utils/hashsum/v1"
const VerifySchema = "missing-utils/hashsum-verify/v1"
const maxManifestBytes = 64 << 20

type Manifest struct {
	Schema     string      `json:"schema"`
	Algorithms []string    `json:"algorithms"`
	Root       string      `json:"root"`
	Files      []File      `json:"files"`
	Errors     []FileError `json:"errors,omitempty"`
}

type File struct {
	Path    string            `json:"path"`
	Size    int64             `json:"size"`
	Mtime   time.Time         `json:"mtime"`
	Status  string            `json:"status"`
	Digests map[string]string `json:"digests"`
}

type FileError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}
type VerifyRecord struct {
	Schema      string           `json:"schema"`
	Path        string           `json:"path"`
	Status      string           `json:"status"`
	Diagnostics []cli.Diagnostic `json:"diagnostics,omitempty"`
}

type CreateOptions struct {
	Root      string
	Paths     []string
	FromJwalk bool
	Workers   int
	Progress  io.Writer
}
type VerifyOptions struct {
	Root              string
	ManifestPath      string
	IncludeUnexpected bool
}

func Create(options CreateOptions, input io.Reader) (Manifest, error) {
	root, err := absoluteRoot(options.Root)
	if err != nil {
		return Manifest{}, err
	}
	paths := append([]string(nil), options.Paths...)
	if options.FromJwalk {
		paths, err = pathsFromJwalk(input)
		if err != nil {
			return Manifest{}, err
		}
	}
	if len(paths) == 0 {
		return Manifest{}, cli.NewError(cli.ExitUsage, "at least one path is required")
	}
	workers := options.Workers
	if workers < 1 {
		workers = 1
	}
	if workers > 64 {
		return Manifest{}, cli.NewError(cli.ExitUsage, "--workers must be between 1 and 64")
	}
	manifest := Manifest{Schema: Schema, Algorithms: []string{"sha256", "blake3"}, Root: root}
	type job struct{ path string }
	jobs := make(chan job)
	results := make(chan File, len(paths))
	errors := make(chan FileError, len(paths))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				file, fileErr := hashFile(root, item.path)
				if fileErr != nil {
					errors <- FileError{Path: item.path, Error: fileErr.Error()}
				} else {
					results <- file
				}
			}
		}()
	}
	go func() {
		for _, path := range paths {
			jobs <- job{path: path}
		}
		close(jobs)
		wg.Wait()
		close(results)
		close(errors)
	}()
	for results != nil || errors != nil {
		select {
		case file, ok := <-results:
			if !ok {
				results = nil
			} else {
				manifest.Files = append(manifest.Files, file)
				if options.Progress != nil {
					fmt.Fprintf(options.Progress, "hashed %s\n", file.Path)
				}
			}
		case fileErr, ok := <-errors:
			if !ok {
				errors = nil
			} else {
				manifest.Errors = append(manifest.Errors, fileErr)
				if options.Progress != nil {
					fmt.Fprintf(options.Progress, "failed %s\n", fileErr.Path)
				}
			}
		}
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	sort.Slice(manifest.Errors, func(i, j int) bool { return manifest.Errors[i].Path < manifest.Errors[j].Path })
	return manifest, nil
}

func hashFile(root, input string) (File, error) {
	abs, rel, err := withinRoot(root, input)
	if err != nil {
		return File{}, err
	}
	pre, err := os.Lstat(abs)
	if err != nil {
		return File{}, err
	}
	if !pre.Mode().IsRegular() {
		return File{}, fmt.Errorf("not a regular file")
	}
	handle, err := os.Open(abs)
	if err != nil {
		return File{}, err
	}
	defer handle.Close()
	sha, blake := sha256.New(), blake3.New()
	if _, err := io.Copy(io.MultiWriter(sha, blake), handle); err != nil {
		return File{}, err
	}
	post, err := os.Lstat(abs)
	if err != nil {
		return File{}, err
	}
	status := "stable"
	if pre.Size() != post.Size() || !pre.ModTime().Equal(post.ModTime()) {
		status = "unstable"
	}
	return File{Path: rel, Size: pre.Size(), Mtime: pre.ModTime().UTC(), Status: status, Digests: map[string]string{"sha256": hex.EncodeToString(sha.Sum(nil)), "blake3": hex.EncodeToString(blake.Sum(nil))}}, nil
}

func Verify(options VerifyOptions) ([]VerifyRecord, error) {
	root, err := absoluteRoot(options.Root)
	if err != nil {
		return nil, err
	}
	manifest, err := readManifest(options.ManifestPath)
	if err != nil {
		return nil, err
	}
	if manifest.Schema != Schema {
		return nil, cli.NewError(cli.ExitUsage, "unsupported manifest schema %q", manifest.Schema)
	}
	for _, algorithm := range manifest.Algorithms {
		if algorithm != "sha256" && algorithm != "blake3" {
			return nil, cli.NewError(cli.ExitUsage, "unknown digest algorithm %q", algorithm)
		}
	}
	seen := map[string]bool{}
	records := make([]VerifyRecord, 0, len(manifest.Files))
	for _, expected := range manifest.Files {
		normalized, err := normalizePath(expected.Path)
		if err != nil {
			return nil, cli.NewError(cli.ExitUsage, "manifest path %q: %v", expected.Path, err)
		}
		if seen[normalized] {
			return nil, cli.NewError(cli.ExitUsage, "duplicate manifest path %q", normalized)
		}
		seen[normalized] = true
		_, _, err = withinRoot(root, normalized)
		if err != nil {
			return nil, cli.NewError(cli.ExitUsage, "manifest path %q: %v", expected.Path, err)
		}
		current, err := hashFile(root, normalized)
		if os.IsNotExist(err) {
			records = append(records, verifyRecord(normalized, "missing", err))
			continue
		}
		if err != nil {
			records = append(records, verifyRecord(normalized, "unreadable", err))
			continue
		}
		if current.Status == "unstable" {
			records = append(records, verifyRecord(normalized, "unstable", nil))
			continue
		}
		if !sameDigests(expected.Digests, current.Digests) {
			records = append(records, verifyRecord(normalized, "changed", nil))
			continue
		}
		records = append(records, verifyRecord(normalized, "ok", nil))
	}
	if options.IncludeUnexpected {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || !entry.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if !seen[rel] {
				records = append(records, verifyRecord(rel, "unexpected", nil))
			}
			return nil
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	return records, nil
}

func verifyRecord(path, status string, err error) VerifyRecord {
	r := VerifyRecord{Schema: VerifySchema, Path: path, Status: status}
	if err != nil {
		r.Diagnostics = []cli.Diagnostic{{Code: status, Message: err.Error(), Path: path}}
	}
	return r
}

func pathsFromJwalk(input io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	var paths []string
	for scanner.Scan() {
		var record struct {
			Path   string  `json:"path"`
			Status string  `json:"status"`
			Type   *string `json:"type"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, cli.NewError(cli.ExitUsage, "invalid jwalk NDJSON: %v", err)
		}
		if record.Path == "" {
			return nil, cli.NewError(cli.ExitUsage, "jwalk NDJSON record has empty path")
		}
		if record.Status != "ok" {
			continue
		}
		if record.Type == nil || *record.Type != "file" {
			continue
		}
		paths = append(paths, record.Path)
	}
	if err := scanner.Err(); err != nil {
		return nil, cli.NewError(cli.ExitUsage, "reading jwalk NDJSON: %v", err)
	}
	return paths, nil
}

func readManifest(path string) (Manifest, error) {
	handle, err := os.Open(path)
	if err != nil {
		return Manifest{}, cli.NewError(cli.ExitRuntime, "opening manifest: %v", err)
	}
	defer handle.Close()
	limited := io.LimitReader(handle, maxManifestBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Manifest{}, cli.NewError(cli.ExitRuntime, "reading manifest: %v", err)
	}
	if len(data) > maxManifestBytes {
		return Manifest{}, cli.NewError(cli.ExitUsage, "manifest exceeds 64 MiB")
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, cli.NewError(cli.ExitUsage, "invalid manifest: %v", err)
	}
	return manifest, nil
}

func WriteManifest(path string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return cli.NewError(cli.ExitRuntime, "encoding manifest: %v", err)
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".hashsum-")
	if err != nil {
		return cli.NewError(cli.ExitRuntime, "create temporary manifest: %v", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return cli.NewError(cli.ExitRuntime, "write manifest: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return cli.NewError(cli.ExitRuntime, "replace manifest: %v", err)
	}
	return nil
}

func absoluteRoot(root string) (string, error) {
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", cli.NewError(cli.ExitRuntime, "get working directory: %v", err)
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", cli.NewError(cli.ExitUsage, "invalid root: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", cli.NewError(cli.ExitUsage, "root %q: %v", root, err)
	}
	return resolved, nil
}

func normalizePath(path string) (string, error) {
	path = filepath.ToSlash(path)
	if path == "" || filepath.IsAbs(path) {
		return "", fmt.Errorf("must be a non-empty relative path")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("escapes root")
	}
	return clean, nil
}

func withinRoot(root, path string) (string, string, error) {
	if filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", "", err
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", "", fmt.Errorf("escapes root")
		}
		path = rel
	}
	normalized, err := normalizePath(path)
	if err != nil {
		return "", "", err
	}
	candidate := filepath.Join(root, filepath.FromSlash(normalized))
	lexical, err := filepath.Rel(root, candidate)
	if err != nil || lexical == ".." || strings.HasPrefix(lexical, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("escapes root")
	}
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", "", fmt.Errorf("resolves outside root")
		}
	}
	return candidate, normalized, nil
}

func sameDigests(expected, actual map[string]string) bool {
	if len(expected) == 0 {
		return false
	}
	for algorithm, value := range expected {
		if actual[algorithm] != value {
			return false
		}
	}
	return true
}
