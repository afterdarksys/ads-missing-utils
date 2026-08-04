package jsondiff

import (
	"strings"
	"testing"
)

func TestCompareSupportsIgnoreAndSetArrays(t *testing.T) {
	left, _ := Decode(strings.NewReader(`{"a":1,"tags":["a","b"]}`))
	right, _ := Decode(strings.NewReader(`{"a":2,"tags":["b","a"]}`))
	report := Compare(left, right, Options{Ignore: map[string]bool{"/a": true}, SetPaths: map[string]bool{"/tags": true}})
	if report.Outcome != "pass" {
		t.Fatalf("report=%#v", report)
	}
}
func TestCompareReportsTypeChange(t *testing.T) {
	left, _ := Decode(strings.NewReader(`{"a":1}`))
	right, _ := Decode(strings.NewReader(`{"a":"1"}`))
	report := Compare(left, right, Options{})
	if len(report.Changes) != 1 || report.Changes[0].Kind != "type_changed" {
		t.Fatalf("report=%#v", report)
	}
}
