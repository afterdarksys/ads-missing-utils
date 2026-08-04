package jsongate

import "testing"

func TestEvaluate(t *testing.T) {
	result := Evaluate([]Finding{{Code: "a", Severity: "high"}, {Code: "b", Severity: "low"}}, Policy{DenyAt: "high"})
	if result.Decision != "deny" || len(result.Reasons) != 1 {
		t.Fatalf("%#v", result)
	}
}
