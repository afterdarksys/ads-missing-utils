// Package jsondiff compares JSON documents using JSON Pointer paths.
package jsondiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

const Schema = "missing-utils/jsondiff/v1"
const maxDocumentBytes = 64 << 20

type Change struct {
	Schema   string `json:"schema"`
	Status   string `json:"status"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Desired  any    `json:"desired,omitempty"`
	Observed any    `json:"observed,omitempty"`
}
type Report struct {
	Schema  string   `json:"schema"`
	Outcome string   `json:"outcome"`
	Changes []Change `json:"changes"`
}
type Options struct {
	Ignore           map[string]bool
	SetPaths         map[string]bool
	NumericTolerance float64
}

func DecodeFile(path string) (any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, cli.NewError(cli.ExitRuntime, "open %q: %v", path, err)
	}
	defer file.Close()
	return Decode(file)
}
func Decode(input io.Reader) (any, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxDocumentBytes+1))
	if err != nil {
		return nil, cli.NewError(cli.ExitRuntime, "read JSON: %v", err)
	}
	if len(data) > maxDocumentBytes {
		return nil, cli.NewError(cli.ExitUsage, "JSON document exceeds 64 MiB input limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, cli.NewError(cli.ExitUsage, "invalid JSON: %v", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, cli.NewError(cli.ExitUsage, "JSON input must contain one document")
	}
	return value, nil
}
func Compare(desired, observed any, options Options) Report {
	report := Report{Schema: Schema, Outcome: "pass"}
	diff(&report.Changes, "", desired, observed, options)
	sort.Slice(report.Changes, func(i, j int) bool { return report.Changes[i].Path < report.Changes[j].Path })
	if len(report.Changes) > 0 {
		report.Outcome = "fail"
	}
	return report
}

func diff(changes *[]Change, path string, desired, observed any, options Options) {
	if options.Ignore[path] {
		return
	}
	if desired == nil && observed != nil {
		add(changes, path, "removed", desired, observed)
		return
	}
	if desired != nil && observed == nil {
		add(changes, path, "added", desired, observed)
		return
	}
	switch expected := desired.(type) {
	case map[string]any:
		actual, ok := observed.(map[string]any)
		if !ok {
			add(changes, path, "type_changed", desired, observed)
			return
		}
		keys := map[string]bool{}
		for key := range expected {
			keys[key] = true
		}
		for key := range actual {
			keys[key] = true
		}
		sorted := make([]string, 0, len(keys))
		for key := range keys {
			sorted = append(sorted, key)
		}
		sort.Strings(sorted)
		for _, key := range sorted {
			diff(changes, pointer(path, key), expected[key], actual[key], options)
		}
	case []any:
		actual, ok := observed.([]any)
		if !ok {
			add(changes, path, "type_changed", desired, observed)
			return
		}
		if options.SetPaths[path] {
			diffSet(changes, path, expected, actual)
			return
		}
		max := len(expected)
		if len(actual) > max {
			max = len(actual)
		}
		for index := 0; index < max; index++ {
			var left, right any
			if index < len(expected) {
				left = expected[index]
			}
			if index < len(actual) {
				right = actual[index]
			}
			diff(changes, pointer(path, strconv.Itoa(index)), left, right, options)
		}
	case json.Number:
		actual, ok := observed.(json.Number)
		if !ok {
			add(changes, path, "type_changed", desired, observed)
			return
		}
		left, lerr := expected.Float64()
		right, rerr := actual.Float64()
		if lerr != nil || rerr != nil || math.Abs(left-right) > options.NumericTolerance {
			add(changes, path, "changed", desired, observed)
		}
	default:
		if fmt.Sprintf("%T", desired) != fmt.Sprintf("%T", observed) {
			add(changes, path, "type_changed", desired, observed)
		} else if !equal(desired, observed) {
			add(changes, path, "changed", desired, observed)
		}
	}
}
func diffSet(changes *[]Change, path string, desired, observed []any) {
	left, right := map[string]any{}, map[string]any{}
	for _, v := range desired {
		left[canonical(v)] = v
	}
	for _, v := range observed {
		right[canonical(v)] = v
	}
	for key, v := range left {
		if _, ok := right[key]; !ok {
			add(changes, path, "added", v, nil)
		}
	}
	for key, v := range right {
		if _, ok := left[key]; !ok {
			add(changes, path, "removed", nil, v)
		}
	}
}
func add(changes *[]Change, path, kind string, desired, observed any) {
	*changes = append(*changes, Change{Schema: Schema, Status: "changed", Path: displayPath(path), Kind: kind, Desired: desired, Observed: observed})
}
func pointer(path, part string) string {
	return path + "/" + strings.ReplaceAll(strings.ReplaceAll(part, "~", "~0"), "/", "~1")
}
func displayPath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}
func equal(left, right any) bool { return canonical(left) == canonical(right) }
func canonical(value any) string { data, _ := json.Marshal(value); return string(data) }
