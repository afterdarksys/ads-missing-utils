package jsonprobe

import (
	"context"
	"strings"
	"testing"
)

func TestFileCheck(t *testing.T) {
	spec, err := Decode(strings.NewReader(`{"checks":[{"name":"go.mod","type":"file","path":"../../go.mod","exists":true}]}`))
	if err != nil {
		t.Fatal(err)
	}
	result := Run(context.Background(), spec)
	if result.Outcome != "pass" || result.Checks[0].Status != "pass" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
func TestRejectsShellLikeType(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"checks":[{"name":"bad","type":"command","path":"whoami"}]}`))
	if err == nil {
		t.Fatal("expected validation error")
	}
}
