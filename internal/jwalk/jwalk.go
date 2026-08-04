package jwalk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/filemeta"
)

const Schema = "missing-utils/jwalk/v1"

type Options struct {
	Roots            []string
	Types            map[string]bool
	Include, Exclude *regexp.Regexp
	MinSize, MaxSize *int64
	OlderThan        *time.Duration
	NewerThan        *time.Duration
	FollowSymlinks   bool
	FailOnError      bool
}

type Record struct {
	Schema       string          `json:"schema"`
	Status       string          `json:"status"`
	Path         string          `json:"path"`
	RelativePath *string         `json:"relative_path"`
	Type         *string         `json:"type"`
	Size         *int64          `json:"size"`
	Mtime        *time.Time      `json:"mtime"`
	Mode         *string         `json:"mode"`
	UID          *uint32         `json:"uid"`
	GID          *uint32         `json:"gid"`
	Inode        *uint64         `json:"inode"`
	Device       *uint64         `json:"device"`
	Error        *cli.Diagnostic `json:"error,omitempty"`
}

func ParseSize(input string) (int64, error) {
	units := map[string]int64{"b": 1, "kib": 1 << 10, "mib": 1 << 20, "gib": 1 << 30}
	lower := strings.ToLower(strings.TrimSpace(input))
	for suffix, multiplier := range units {
		if strings.HasSuffix(lower, suffix) {
			value := strings.TrimSuffix(lower, suffix)
			var n int64
			if _, err := fmt.Sscan(value, &n); err != nil || n < 0 {
				return 0, fmt.Errorf("invalid size %q", input)
			}
			return n * multiplier, nil
		}
	}
	var n int64
	if _, err := fmt.Sscan(lower, &n); err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q", input)
	}
	return n, nil
}

func Walk(options Options) ([]Record, error) {
	if len(options.Roots) == 0 {
		return nil, cli.NewError(cli.ExitUsage, "at least one root path is required")
	}
	now := time.Now()
	var records []Record
	pending := append([]string(nil), options.Roots...)
	visitedDirectories := map[string]bool{}
	for len(pending) > 0 {
		root := pending[0]
		pending = pending[1:]
		info, err := os.Lstat(root)
		if err != nil {
			return nil, cli.NewError(cli.ExitUsage, "root %q: %v", root, err)
		}
		if info.IsDir() {
			resolved, err := filepath.EvalSymlinks(root)
			if err != nil {
				return nil, cli.NewError(cli.ExitUsage, "resolve root %q: %v", root, err)
			}
			if visitedDirectories[resolved] {
				continue
			}
			visitedDirectories[resolved] = true
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				records = append(records, errorRecord(path, walkErr))
				if options.FailOnError {
					return walkErr
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if !options.FollowSymlinks {
					return nil
				}
				if target, targetErr := filepath.EvalSymlinks(path); targetErr == nil {
					if targetInfo, statErr := os.Stat(target); statErr == nil && targetInfo.IsDir() {
						pending = append(pending, target)
					}
				}
			}
			info, err := entry.Info()
			if err != nil {
				records = append(records, errorRecord(path, err))
				if options.FailOnError {
					return err
				}
				return nil
			}
			if info.IsDir() && options.FollowSymlinks {
				if resolved, resolveErr := filepath.EvalSymlinks(path); resolveErr == nil {
					visitedDirectories[resolved] = true
				}
			}
			typeName := entryType(info.Mode())
			if !matches(path, root, info, typeName, options, now) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			meta := filemeta.FromInfo(info)
			size, mtime, typ, mode := info.Size(), info.ModTime().UTC(), typeName, meta.Mode
			relCopy := filepath.ToSlash(rel)
			records = append(records, Record{Schema: Schema, Status: "ok", Path: path, RelativePath: &relCopy, Type: &typ, Size: &size, Mtime: &mtime, Mode: &mode, UID: meta.UID, GID: meta.GID, Inode: meta.Inode, Device: meta.Device})
			return nil
		})
		if err != nil && options.FailOnError {
			return records, cli.NewError(cli.ExitPartial, "walking %q: %v", root, err)
		}
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	return records, nil
}

func errorRecord(path string, err error) Record {
	return Record{Schema: Schema, Status: "error", Path: path, Error: &cli.Diagnostic{Code: "walk_error", Message: err.Error(), Path: path}}
}

func entryType(mode os.FileMode) string {
	switch {
	case mode.IsRegular():
		return "file"
	case mode.IsDir():
		return "directory"
	case mode&os.ModeSymlink != 0:
		return "symlink"
	default:
		return "other"
	}
}

func matches(path, root string, info os.FileInfo, typeName string, options Options, now time.Time) bool {
	if len(options.Types) > 0 && !options.Types[typeName] {
		return false
	}
	rel, _ := filepath.Rel(root, path)
	name := filepath.ToSlash(rel)
	if options.Include != nil && !options.Include.MatchString(name) {
		return false
	}
	if options.Exclude != nil && options.Exclude.MatchString(name) {
		return false
	}
	if options.MinSize != nil && info.Size() < *options.MinSize {
		return false
	}
	if options.MaxSize != nil && info.Size() > *options.MaxSize {
		return false
	}
	age := now.Sub(info.ModTime())
	if options.OlderThan != nil && age < *options.OlderThan {
		return false
	}
	if options.NewerThan != nil && age > *options.NewerThan {
		return false
	}
	return true
}
