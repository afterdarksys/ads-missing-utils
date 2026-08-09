package macossec

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type CloneCandidate struct {
	Path    string   `json:"path"`
	SHA256  string   `json:"sha256"`
	Size    int64    `json:"size"`
	Matches []string `json:"matches"`
}
type CLTReport struct {
	Schema     string           `json:"schema"`
	Root       string           `json:"root"`
	Candidates []CloneCandidate `json:"candidates"`
	Findings   []Finding        `json:"findings,omitempty"`
}

func ScanCloneExecutables(root string) (CLTReport, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return CLTReport{}, err
	}
	type item struct {
		path, hash         string
		size               int64
		hidden, executable bool
	}
	items := []item{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || path == root {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		digest, err := hashFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		items = append(items, item{filepath.ToSlash(rel), digest, info.Size(), strings.HasPrefix(filepath.Base(path), "."), info.Mode().Perm()&0o111 != 0})
		return nil
	})
	if err != nil {
		return CLTReport{}, err
	}
	groups := map[string][]item{}
	for _, v := range items {
		key := fmt.Sprintf("%d:%s", v.size, v.hash)
		groups[key] = append(groups[key], v)
	}
	report := CLTReport{Schema: "missing-utils/clt/v1", Root: root, Candidates: []CloneCandidate{}}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		paths := []string{}
		for _, v := range group {
			paths = append(paths, v.path)
		}
		sort.Strings(paths)
		for _, v := range group {
			if v.hidden && v.executable {
				matches := []string{}
				for _, p := range paths {
					if p != v.path {
						matches = append(matches, p)
					}
				}
				report.Candidates = append(report.Candidates, CloneCandidate{v.path, v.hash, v.size, matches})
			}
		}
	}
	sort.Slice(report.Candidates, func(i, j int) bool { return report.Candidates[i].Path < report.Candidates[j].Path })
	return report, nil
}
func QuarantineCandidates(report CLTReport, dest string) ([]string, error) {
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return nil, err
	}
	moved := []string{}
	for _, candidate := range report.Candidates {
		source := safeJoin(report.Root, candidate.Path)
		if source == "" {
			continue
		}
		name := filepath.Base(source) + "." + candidate.SHA256[:8]
		target := filepath.Join(dest, name)
		if _, err := os.Lstat(target); err == nil {
			return moved, fmt.Errorf("quarantine collision: %s", target)
		}
		if err := os.Rename(source, target); err != nil {
			return moved, err
		}
		moved = append(moved, target)
	}
	return moved, nil
}

type SignedBaseline struct {
	Schema     string              `json:"schema"`
	Root       string              `json:"root"`
	CreatedAt  string              `json:"created_at"`
	Entries    map[string]string   `json:"entries"`
	ScanErrors []BaselineScanError `json:"scan_errors,omitempty"`
	Signature  string              `json:"signature"`
}
type BaselineScanError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}
type DTPChange struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

func CreateDaemonBaseline(root string, key []byte) (SignedBaseline, error) {
	if err := validateBaselineKey(key); err != nil {
		return SignedBaseline{}, err
	}
	entries, scanErrors, err := executableHashes(root)
	if err != nil {
		return SignedBaseline{}, err
	}
	b := SignedBaseline{Schema: "missing-utils/dtp-state/v1", Root: root, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Entries: entries, ScanErrors: scanErrors}
	b.Signature, err = signBaseline(b, key)
	if err != nil {
		return SignedBaseline{}, err
	}
	return b, nil
}
func CheckDaemonBaseline(b SignedBaseline, key []byte) ([]DTPChange, error) {
	if err := validateBaselineKey(key); err != nil {
		return nil, err
	}
	expectedSignature, err := signBaseline(b, key)
	if err != nil {
		return nil, err
	}
	if !hmac.Equal([]byte(b.Signature), []byte(expectedSignature)) {
		return nil, fmt.Errorf("baseline signature verification failed")
	}
	if len(b.ScanErrors) > 0 {
		return nil, fmt.Errorf("baseline contains unreadable paths: %s", formatBaselineScanErrors(b.ScanErrors))
	}
	live, scanErrors, err := executableHashes(b.Root)
	if err != nil {
		return nil, err
	}
	if len(scanErrors) > 0 {
		return nil, fmt.Errorf("daemon scan found unreadable paths: %s", formatBaselineScanErrors(scanErrors))
	}
	changes := []DTPChange{}
	for path, digest := range b.Entries {
		now, ok := live[path]
		if !ok {
			changes = append(changes, DTPChange{"removed", path})
		} else if now != digest {
			changes = append(changes, DTPChange{"modified", path})
		}
	}
	for path := range live {
		if _, ok := b.Entries[path]; !ok {
			changes = append(changes, DTPChange{"added", path})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes, nil
}
func SaveBaseline(path string, b SignedBaseline) error { return atomicJSON(path, b) }
func LoadBaseline(path string) (SignedBaseline, error) {
	var b SignedBaseline
	data, err := os.ReadFile(path)
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(data, &b)
	return b, err
}

type Process struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
}
type TransientJob struct {
	PID               int    `json:"pid"`
	Command           string `json:"command"`
	FirstSeen         string `json:"first_seen"`
	LastSeen          string `json:"last_seen"`
	ExecutableDeleted bool   `json:"executable_deleted"`
}
type Tracker struct{ active map[int]tracked }
type tracked struct {
	process     Process
	first, last time.Time
}

func NewTracker() *Tracker { return &Tracker{active: map[int]tracked{}} }
func (t *Tracker) Observe(now time.Time, current []Process) []TransientJob {
	seen := map[int]bool{}
	for _, p := range current {
		seen[p.PID] = true
		if old, ok := t.active[p.PID]; ok {
			old.last = now
			old.process = p
			t.active[p.PID] = old
		} else {
			t.active[p.PID] = tracked{p, now, now}
		}
	}
	ended := []TransientJob{}
	for pid, v := range t.active {
		if !seen[pid] {
			_, err := os.Stat(v.process.Command)
			ended = append(ended, TransientJob{pid, v.process.Command, v.first.UTC().Format(time.RFC3339Nano), v.last.UTC().Format(time.RFC3339Nano), os.IsNotExist(err)})
			delete(t.active, pid)
		}
	}
	sort.Slice(ended, func(i, j int) bool { return ended[i].PID < ended[j].PID })
	return ended
}
func SnapshotProcesses() ([]Process, error) {
	out, err := exec.Command("ps", "-axo", "pid=,comm=").Output()
	if err != nil {
		return nil, err
	}
	result := []Process{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err == nil {
			result = append(result, Process{pid, strings.Join(fields[1:], " ")})
		}
	}
	return result, scanner.Err()
}

type SecretMatch struct {
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
}

var secretPatterns = []struct {
	name string
	re   *regexp.Regexp
}{{"private_key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)}, {"aws_access_key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)}, {"github_token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{20,}\b`)}, {"seed_phrase", regexp.MustCompile(`(?i)\b(?:seed phrase|mnemonic)\s*[:=]\s*[a-z]{3,12}(?:\s+[a-z]{3,12}){11,23}\b`)}}

func DetectSecrets(content string) []SecretMatch {
	result := []SecretMatch{}
	for _, pattern := range secretPatterns {
		if match := pattern.re.FindString(content); match != "" {
			sum := sha256.Sum256([]byte(match))
			result = append(result, SecretMatch{pattern.name, hex.EncodeToString(sum[:])[:12]})
		}
	}
	return result
}

type PermissionNode struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}
type PermissionEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
}
type IntentGraph struct {
	Schema   string           `json:"schema"`
	Nodes    []PermissionNode `json:"nodes"`
	Edges    []PermissionEdge `json:"edges"`
	Findings []Finding        `json:"findings"`
}

func MapAppleScriptIntent(script string) IntentGraph {
	graph := IntentGraph{Schema: "missing-utils/amtm/v1", Nodes: []PermissionNode{{"script", "automation"}}, Edges: []PermissionEdge{}, Findings: []Finding{}}
	rules := []struct {
		pattern, kind, reason, code string
		severity                    string
	}{{`(?i)do\s+shell\s+script`, "shell", "executes a shell command", "AMTM_SHELL", "high"}, {`(?i)system\s+events|keystroke|key code`, "accessibility", "controls UI input", "AMTM_ACCESSIBILITY", "high"}, {`(?i)the clipboard|set the clipboard`, "clipboard", "reads or writes clipboard", "AMTM_CLIPBOARD", "medium"}, {`(?i)curl\s|https?://`, "network", "references network access", "AMTM_NETWORK", "medium"}, {`(?i)delete\s+file|move\s+file|set eof|open for access`, "filesystem", "mutates filesystem content", "AMTM_FILESYSTEM", "medium"}}
	seen := map[string]bool{}
	for _, rule := range rules {
		if matched, _ := regexp.MatchString(rule.pattern, script); matched && !seen[rule.kind] {
			seen[rule.kind] = true
			graph.Nodes = append(graph.Nodes, PermissionNode{rule.kind, "permission"})
			graph.Edges = append(graph.Edges, PermissionEdge{"script", rule.kind, rule.reason})
			graph.Findings = append(graph.Findings, Finding{rule.code, rule.severity, "", rule.reason})
		}
	}
	return graph
}

type AccessibilityGrant struct {
	Client        string `json:"client"`
	Authorization int    `json:"authorization"`
	LastModified  int64  `json:"last_modified,omitempty"`
}

func AuditAccessibility(dbPath string) ([]AccessibilityGrant, error) {
	sqlite, err := exec.LookPath("sqlite3")
	if err != nil {
		return nil, fmt.Errorf("sqlite3 is required: %w", err)
	}
	query := "SELECT client,auth_value,last_modified FROM access WHERE service='kTCCServiceAccessibility' ORDER BY client"
	out, err := exec.Command(sqlite, "-readonly", "-separator", "|", dbPath, query).Output()
	if err != nil {
		return nil, fmt.Errorf("read TCC database (Full Disk Access may be required): %w", err)
	}
	return parseAccessibilityGrants(string(out))
}

func parseAccessibilityGrants(output string) ([]AccessibilityGrant, error) {
	grants := []AccessibilityGrant{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "|")
		if len(parts) < 2 {
			continue
		}
		auth, _ := strconv.Atoi(parts[1])
		modified := int64(0)
		if len(parts) > 2 {
			modified, _ = strconv.ParseInt(parts[2], 10, 64)
		}
		grants = append(grants, AccessibilityGrant{parts[0], auth, modified})
	}
	return grants, scanner.Err()
}

type WebKitReport struct {
	Schema       string    `json:"schema"`
	Root         string    `json:"root"`
	Findings     []Finding `json:"findings"`
	FilesScanned int       `json:"files_scanned"`
}

func ScanWebKit(root string) (WebKitReport, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return WebKitReport{}, err
	}
	report := WebKitReport{Schema: "missing-utils/webkit-tool/v1", Root: root, Findings: []Finding{}}
	patterns := []struct {
		re                      *regexp.Regexp
		code, severity, message string
	}{{regexp.MustCompile(`(?i)allowUniversalAccessFromFileURLs`), "WEBKIT_UNIVERSAL_FILE_ACCESS", "high", "universal access from file URLs"}, {regexp.MustCompile(`(?i)allowFileAccessFromFileURLs`), "WEBKIT_FILE_ACCESS", "medium", "file access from file URLs"}, {regexp.MustCompile(`(?i)setValue\s*\([^\n]+forKey:\s*["'](?:allowUniversalAccess|allowFileAccess)`), "WEBKIT_PRIVATE_KVC", "high", "private WebKit preference set through KVC"}, {regexp.MustCompile(`(?i)javaScriptCanOpenWindowsAutomatically\s*=\s*true`), "WEBKIT_JS_POPUPS", "low", "JavaScript automatic windows enabled"}, {regexp.MustCompile(`(?i)addScriptMessageHandler|add\([^\n]+name:`), "WEBKIT_SCRIPT_BRIDGE", "medium", "native script-message bridge exposed"}}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > 2<<20 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".swift" && ext != ".m" && ext != ".mm" && ext != ".js" && ext != ".html" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		report.FilesScanned++
		rel, _ := filepath.Rel(root, path)
		for _, pattern := range patterns {
			if pattern.re.Match(data) {
				report.Findings = append(report.Findings, Finding{pattern.code, pattern.severity, filepath.ToSlash(rel), pattern.message})
			}
		}
		return nil
	})
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Path == report.Findings[j].Path {
			return report.Findings[i].Code < report.Findings[j].Code
		}
		return report.Findings[i].Path < report.Findings[j].Path
	})
	return report, err
}

func executableHashes(root string) (map[string]string, []BaselineScanError, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	entries := map[string]string{}
	scanErrors := []BaselineScanError{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr
			}
			scanErrors = append(scanErrors, baselineScanError(root, path, walkErr))
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			scanErrors = append(scanErrors, baselineScanError(root, path, err))
			return nil
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			return nil
		}
		digest, err := hashFile(path)
		if err != nil {
			scanErrors = append(scanErrors, baselineScanError(root, path, err))
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		entries[filepath.ToSlash(rel)] = digest
		return nil
	})
	sort.Slice(scanErrors, func(i, j int) bool { return scanErrors[i].Path < scanErrors[j].Path })
	return entries, scanErrors, err
}
func signBaseline(b SignedBaseline, key []byte) (string, error) {
	clone := b
	clone.Signature = ""
	data, err := json.Marshal(clone)
	if err != nil {
		return "", fmt.Errorf("marshal baseline for signing: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(data); err != nil {
		return "", fmt.Errorf("sign baseline: %w", err)
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}
func validateBaselineKey(key []byte) error {
	if len(key) < 16 {
		return fmt.Errorf("key must contain at least 16 bytes")
	}
	return nil
}
func baselineScanError(root, path string, err error) BaselineScanError {
	rel, relErr := filepath.Rel(root, path)
	if relErr != nil {
		rel = path
	}
	return BaselineScanError{Path: filepath.ToSlash(rel), Message: err.Error()}
}
func formatBaselineScanErrors(scanErrors []BaselineScanError) string {
	items := make([]string, 0, len(scanErrors))
	for _, scanErr := range scanErrors {
		items = append(items, fmt.Sprintf("%s (%s)", scanErr.Path, scanErr.Message))
	}
	return strings.Join(items, "; ")
}
func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	_, err = io.Copy(h, file)
	return hex.EncodeToString(h.Sum(nil)), err
}
func atomicJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-tmp-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	tmp.Chmod(0o600)
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
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
	return os.Rename(name, path)
}
func safeJoin(root, rel string) string {
	if filepath.IsAbs(rel) {
		return ""
	}
	joined := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	check, err := filepath.Rel(root, joined)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return ""
	}
	return joined
}
