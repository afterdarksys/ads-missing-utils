package varmerge

import (
	"github.com/afterdarksys/ads-missing-utils/internal/envsub"
	"testing"
)

func TestMergePrecedenceAndSecretProvenance(t *testing.T) {
	result, err := Merge([]Source{{Name: "file", Values: map[string]any{"PORT": "80", "TOKEN": "first"}}, {Name: "set", Values: map[string]any{"PORT": "443", "TOKEN": "second"}}}, map[string]envsub.Field{"PORT": {Type: "integer"}, "TOKEN": {Secret: true}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Values["PORT"] != "443" || result.Provenance["TOKEN"].Value != "***" || result.Provenance["TOKEN"].Source != "set" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(result.Shadowed["PORT"]) != 1 {
		t.Fatalf("shadowed = %#v", result.Shadowed)
	}
}
