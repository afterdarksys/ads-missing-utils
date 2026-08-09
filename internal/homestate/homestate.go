package homestate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const StateSchema = "missing-utils/homestate-state/v1"

type StateEntry struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	Mode       uint32 `json:"mode"`
	MtimeNS    int64  `json:"mtime_ns"`
	SHA256     string `json:"sha256,omitempty"`
	LinkTarget string `json:"link_target,omitempty"`
	Unstable   bool   `json:"unstable,omitempty"`
}

type RenameRecord struct {
	From       string `json:"from"`
	To         string `json:"to"`
	AppliedAt  string `json:"applied_at"`
	RestoredAt string `json:"restored_at,omitempty"`
}

type DB struct {
	Schema    string                `json:"schema"`
	Root      string                `json:"root"`
	IndexedAt string                `json:"indexed_at"`
	Entries   map[string]StateEntry `json:"entries"`
	Renames   []RenameRecord        `json:"renames,omitempty"`
}

type Change struct {
	Status   string   `json:"status"`
	Path     string   `json:"path"`
	OldPath  string   `json:"old_path,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

type BrokenName struct {
	Path       string   `json:"path"`
	Characters []string `json:"characters"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

func Index(root, statePath string) (DB, error) {
	root, err := normalizeRoot(root)
	if err != nil {
		return DB{}, err
	}
	// Create the storage directory before scanning so a state path beneath the
	// mapped root does not manufacture directory additions immediately after
	// the baseline is written.
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return DB{}, err
	}
	entries, _, err := scan(root, statePath)
	if err != nil {
		return DB{}, err
	}
	db := DB{Schema: StateSchema, Root: root, IndexedAt: time.Now().UTC().Format(time.RFC3339Nano), Entries: entries}
	if old, loadErr := Load(statePath); loadErr == nil && samePath(old.Root, root) {
		db.Renames = old.Renames
	}
	if err := save(statePath, db); err != nil {
		return DB{}, err
	}
	return db, nil
}

func Load(path string) (DB, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DB{}, err
	}
	var db DB
	if err := json.Unmarshal(data, &db); err != nil {
		return DB{}, fmt.Errorf("decode homestate database: %w", err)
	}
	if db.Schema != StateSchema || db.Root == "" || db.Entries == nil {
		return DB{}, fmt.Errorf("unsupported or incomplete homestate database")
	}
	return db, nil
}

func Check(db DB, statePath string) ([]Change, []Diagnostic, error) {
	live, diagnostics, err := scan(db.Root, statePath)
	if err != nil {
		return nil, diagnostics, err
	}
	changes := compare(db.Entries, live)
	return changes, diagnostics, nil
}

func compare(old, live map[string]StateEntry) []Change {
	added, removed := map[string]StateEntry{}, map[string]StateEntry{}
	changes := []Change{}
	for path, before := range old {
		after, ok := live[path]
		if !ok {
			removed[path] = before
			continue
		}
		if !entryEqual(before, after) {
			changes = append(changes, Change{Status: "modified", Path: path, Evidence: entryDiff(before, after)})
		}
	}
	for path, entry := range live {
		if _, ok := old[path]; !ok {
			added[path] = entry
		}
	}
	type group struct{ old, new []string }
	groups := map[string]*group{}
	for path, entry := range removed {
		if entry.Type == "file" && entry.SHA256 != "" {
			key := fmt.Sprintf("%d:%s", entry.Size, entry.SHA256)
			if groups[key] == nil {
				groups[key] = &group{}
			}
			groups[key].old = append(groups[key].old, path)
		}
	}
	for path, entry := range added {
		if entry.Type == "file" && entry.SHA256 != "" {
			key := fmt.Sprintf("%d:%s", entry.Size, entry.SHA256)
			if groups[key] == nil {
				groups[key] = &group{}
			}
			groups[key].new = append(groups[key].new, path)
		}
	}
	for _, g := range groups {
		sort.Strings(g.old)
		sort.Strings(g.new)
		if len(g.old) == 1 && len(g.new) == 1 {
			changes = append(changes, Change{Status: "moved", Path: g.new[0], OldPath: g.old[0], Evidence: []string{"sha256 and size match"}})
			delete(removed, g.old[0])
			delete(added, g.new[0])
		} else if len(g.old) > 0 && len(g.new) > 0 {
			for _, path := range g.new {
				changes = append(changes, Change{Status: "ambiguous_move", Path: path, Evidence: append([]string{"multiple sha256 and size matches"}, g.old...)})
				delete(added, path)
			}
			for _, path := range g.old {
				delete(removed, path)
			}
		}
	}
	for path := range added {
		changes = append(changes, Change{Status: "added", Path: path})
	}
	for path := range removed {
		changes = append(changes, Change{Status: "removed", Path: path})
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Path == changes[j].Path {
			return changes[i].Status < changes[j].Status
		}
		return changes[i].Path < changes[j].Path
	})
	return changes
}

func BrokenNames(root string) ([]BrokenName, error) {
	root, err := normalizeRoot(root)
	if err != nil {
		return nil, err
	}
	result := []BrokenName{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		chars := invalidCharacters(entry.Name())
		if len(chars) > 0 {
			rel, _ := filepath.Rel(root, path)
			result = append(result, BrokenName{Path: filepath.ToSlash(rel), Characters: chars})
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, err
}

func FixNames(root, statePath string) ([]RenameRecord, []Diagnostic, error) {
	root, err := normalizeRoot(root)
	if err != nil {
		return nil, nil, err
	}
	db := DB{Schema: StateSchema, Root: root, IndexedAt: time.Now().UTC().Format(time.RFC3339Nano), Entries: map[string]StateEntry{}}
	if loaded, loadErr := Load(statePath); loadErr == nil {
		if !samePath(loaded.Root, root) {
			return nil, nil, fmt.Errorf("state root %q does not match %q", loaded.Root, root)
		}
		db = loaded
	} else if !errors.Is(loadErr, os.ErrNotExist) {
		return nil, nil, loadErr
	}
	renamed, diagnostics := []RenameRecord{}, []Diagnostic{}
	var visit func(string) error
	visit = func(dir string) error {
		children, readErr := os.ReadDir(dir)
		if readErr != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_READ_FAILED", Message: readErr.Error(), Path: relative(root, dir)})
			return nil
		}
		for _, child := range children {
			current := filepath.Join(dir, child.Name())
			if len(invalidCharacters(child.Name())) > 0 {
				clean := sanitizeName(child.Name())
				target := availableTarget(dir, clean, child.Name())
				if renameErr := os.Rename(current, target); renameErr != nil {
					diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_RENAME_FAILED", Message: renameErr.Error(), Path: relative(root, current)})
					continue
				}
				record := RenameRecord{From: relative(root, current), To: relative(root, target), AppliedAt: time.Now().UTC().Format(time.RFC3339Nano)}
				db.Renames = append(db.Renames, record)
				renamed = append(renamed, record)
				current = target
			}
			info, infoErr := os.Lstat(current)
			if infoErr == nil && info.IsDir() {
				if err := visit(current); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return renamed, diagnostics, err
	}
	if err := save(statePath, db); err != nil {
		return renamed, diagnostics, err
	}
	return renamed, diagnostics, nil
}

func RestoreNames(root, statePath string) ([]RenameRecord, []Diagnostic, error) {
	root, err := normalizeRoot(root)
	if err != nil {
		return nil, nil, err
	}
	db, err := Load(statePath)
	if err != nil {
		return nil, nil, err
	}
	if !samePath(db.Root, root) {
		return nil, nil, fmt.Errorf("state root %q does not match %q", db.Root, root)
	}
	restored, diagnostics := []RenameRecord{}, []Diagnostic{}
	for i := len(db.Renames) - 1; i >= 0; i-- {
		record := &db.Renames[i]
		if record.RestoredAt != "" {
			continue
		}
		from, to := safeJoin(root, record.From), safeJoin(root, record.To)
		if from == "" || to == "" {
			diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_UNSAFE_RESTORE", Message: "rename record escapes root", Path: record.From})
			continue
		}
		if _, statErr := os.Lstat(to); statErr != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_RESTORE_SOURCE_MISSING", Message: statErr.Error(), Path: record.To})
			continue
		}
		if _, statErr := os.Lstat(from); statErr == nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_RESTORE_COLLISION", Message: "original path already exists", Path: record.From})
			continue
		} else if !errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if renameErr := os.Rename(to, from); renameErr != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_RESTORE_FAILED", Message: renameErr.Error(), Path: record.To})
			continue
		}
		record.RestoredAt = time.Now().UTC().Format(time.RFC3339Nano)
		restored = append(restored, *record)
	}
	if err := save(statePath, db); err != nil {
		return restored, diagnostics, err
	}
	return restored, diagnostics, nil
}

func scan(root, excluded string) (map[string]StateEntry, []Diagnostic, error) {
	root, err := normalizeRoot(root)
	if err != nil {
		return nil, nil, err
	}
	excluded, _ = filepath.Abs(excluded)
	entries, diagnostics := map[string]StateEntry{}, []Diagnostic{}
	err = filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_WALK_FAILED", Message: walkErr.Error(), Path: relative(root, path)})
			if dirEntry != nil && dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		absolute, _ := filepath.Abs(path)
		if samePath(absolute, excluded) || strings.HasPrefix(filepath.Base(absolute), ".homestate-tmp-") {
			if dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, infoErr := os.Lstat(path)
		if infoErr != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_STAT_FAILED", Message: infoErr.Error(), Path: relative(root, path)})
			return nil
		}
		entry := StateEntry{Path: relative(root, path), Type: fileType(info.Mode()), Size: info.Size(), Mode: uint32(info.Mode()), MtimeNS: info.ModTime().UnixNano()}
		if info.Mode().IsRegular() {
			entry.SHA256, entry.Unstable, infoErr = hashStable(path, info)
			if infoErr != nil {
				diagnostics = append(diagnostics, Diagnostic{Code: "HOMESTATE_HASH_FAILED", Message: infoErr.Error(), Path: entry.Path})
			}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			entry.LinkTarget, _ = os.Readlink(path)
		}
		entries[entry.Path] = entry
		return nil
	})
	return entries, diagnostics, err
}

func hashStable(path string, before os.FileInfo) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", false, err
	}
	after, err := os.Stat(path)
	if err != nil {
		return "", false, err
	}
	unstable := before.Size() != after.Size() || before.ModTime() != after.ModTime()
	return hex.EncodeToString(h.Sum(nil)), unstable, nil
}

func save(path string, db DB) error {
	if path == "" {
		return fmt.Errorf("state path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".homestate-tmp-")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(db); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func normalizeRoot(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root is not a directory: %s", root)
	}
	return filepath.Clean(root), nil
}
func relative(root, path string) string {
	rel, _ := filepath.Rel(root, path)
	return filepath.ToSlash(rel)
}
func samePath(a, b string) bool {
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	return filepath.Clean(aa) == filepath.Clean(bb)
}
func fileType(mode os.FileMode) string {
	if mode.IsRegular() {
		return "file"
	}
	if mode.IsDir() {
		return "dir"
	}
	if mode&os.ModeSymlink != 0 {
		return "symlink"
	}
	return "other"
}
func entryEqual(a, b StateEntry) bool {
	if a.Type == "dir" && b.Type == "dir" {
		return a.Mode == b.Mode
	}
	return a.Type == b.Type && a.Size == b.Size && a.Mode == b.Mode && a.MtimeNS == b.MtimeNS && a.SHA256 == b.SHA256 && a.LinkTarget == b.LinkTarget && a.Unstable == b.Unstable
}
func entryDiff(a, b StateEntry) []string {
	result := []string{}
	if a.Type != b.Type {
		result = append(result, "type changed")
	}
	if a.Type != "dir" && a.Size != b.Size {
		result = append(result, "size changed")
	}
	if a.Mode != b.Mode {
		result = append(result, "mode changed")
	}
	if a.Type != "dir" && a.MtimeNS != b.MtimeNS {
		result = append(result, "mtime changed")
	}
	if a.SHA256 != b.SHA256 {
		result = append(result, "content hash changed")
	}
	if a.LinkTarget != b.LinkTarget {
		result = append(result, "symlink target changed")
	}
	return result
}
func invalidCharacters(name string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range name {
		bad := unicode.IsControl(r) || strings.ContainsRune("~:;'\"", r)
		if bad {
			s := string(r)
			if unicode.IsControl(r) {
				s = fmt.Sprintf("U+%04X", r)
			}
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	sort.Strings(out)
	return out
}
func sanitizeName(name string) string {
	var b strings.Builder
	invalid := false
	for _, r := range name {
		bad := unicode.IsControl(r) || strings.ContainsRune("~:;'\"", r)
		if bad {
			if !invalid {
				b.WriteByte('_')
			}
			invalid = true
		} else {
			b.WriteRune(r)
			invalid = false
		}
	}
	clean := b.String()
	if clean == "" || clean == "." || clean == ".." {
		return "_"
	}
	return clean
}
func availableTarget(dir, clean, original string) string {
	candidate := filepath.Join(dir, clean)
	if clean == original {
		return candidate
	}
	if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate
	}
	sum := sha256.Sum256([]byte(original))
	suffix := hex.EncodeToString(sum[:])[:8]
	ext := filepath.Ext(clean)
	base := strings.TrimSuffix(clean, ext)
	for i := 0; ; i++ {
		name := fmt.Sprintf("%s_%s%s", base, suffix, ext)
		if i > 0 {
			name = fmt.Sprintf("%s_%s_%d%s", base, suffix, i, ext)
		}
		candidate = filepath.Join(dir, name)
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}
func safeJoin(root, rel string) string {
	if filepath.IsAbs(rel) {
		return ""
	}
	path := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	check, err := filepath.Rel(root, path)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return ""
	}
	return path
}
