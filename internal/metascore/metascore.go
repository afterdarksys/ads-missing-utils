package metascore

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const StateSchema = "missing-utils/metascore-state/v1"

type Entry struct {
	Path    string   `json:"path"`
	Size    int64    `json:"size"`
	Mode    uint32   `json:"mode"`
	MtimeNS int64    `json:"mtime_ns"`
	Xattrs  []string `json:"xattrs,omitempty"`
}
type State struct {
	Schema    string           `json:"schema"`
	Root      string           `json:"root"`
	IndexedAt string           `json:"indexed_at"`
	Entries   map[string]Entry `json:"entries"`
}
type Anomaly struct {
	Path     string   `json:"path"`
	Score    int      `json:"score"`
	Evidence []string `json:"evidence"`
}
type Report struct {
	Schema    string    `json:"schema"`
	Root      string    `json:"root"`
	Anomalies []Anomaly `json:"anomalies"`
}

func Index(root, statePath string) (State, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return State{}, err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return State{}, err
	}
	entries, err := collect(root, statePath)
	if err != nil {
		return State{}, err
	}
	state := State{Schema: StateSchema, Root: root, IndexedAt: time.Now().UTC().Format(time.RFC3339Nano), Entries: entries}
	if err := save(statePath, state); err != nil {
		return State{}, err
	}
	return state, nil
}
func Check(statePath string) (Report, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		return Report{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return Report{}, err
	}
	if state.Schema != StateSchema {
		return Report{}, fmt.Errorf("unsupported metascore state")
	}
	live, err := collect(state.Root, statePath)
	if err != nil {
		return Report{}, err
	}
	report := Report{Schema: "missing-utils/metascore/v1", Root: state.Root, Anomalies: []Anomaly{}}
	for path, old := range state.Entries {
		now, ok := live[path]
		if !ok {
			report.Anomalies = append(report.Anomalies, Anomaly{Path: path, Score: 40, Evidence: []string{"path removed"}})
			continue
		}
		score := 0
		e := []string{}
		if old.Mode != now.Mode {
			score += 30
			e = append(e, "mode changed")
		}
		if old.Size != now.Size {
			score += 20
			e = append(e, "size changed")
		}
		if old.MtimeNS != now.MtimeNS {
			score += 10
			e = append(e, "mtime changed")
		}
		if strings.Join(old.Xattrs, "\x00") != strings.Join(now.Xattrs, "\x00") {
			score += 40
			e = append(e, "extended attribute names changed")
		}
		if score > 0 {
			report.Anomalies = append(report.Anomalies, Anomaly{Path: path, Score: score, Evidence: e})
		}
	}
	for path := range live {
		if _, ok := state.Entries[path]; !ok {
			report.Anomalies = append(report.Anomalies, Anomaly{Path: path, Score: 10, Evidence: []string{"path added"}})
		}
	}
	sort.Slice(report.Anomalies, func(i, j int) bool { return report.Anomalies[i].Path < report.Anomalies[j].Path })
	return report, nil
}
func collect(root, excluded string) (map[string]Entry, error) {
	entries := map[string]Entry{}
	excluded, _ = filepath.Abs(excluded)
	err := filepath.WalkDir(root, func(path string, de fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		absolute, _ := filepath.Abs(path)
		if absolute == excluded || strings.HasPrefix(filepath.Base(path), ".metascore-tmp-") {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		entries[filepath.ToSlash(rel)] = Entry{Path: filepath.ToSlash(rel), Size: info.Size(), Mode: uint32(info.Mode()), MtimeNS: info.ModTime().UnixNano(), Xattrs: xattrs(path)}
		return nil
	})
	return entries, err
}

func save(path string, state State) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".metascore-tmp-")
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
	if err := encoder.Encode(state); err != nil {
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
func xattrs(path string) []string {
	binary, err := exec.LookPath("xattr")
	if err != nil {
		return nil
	}
	out, err := exec.Command(binary, path).Output()
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(out))
	sort.Strings(fields)
	return fields
}
