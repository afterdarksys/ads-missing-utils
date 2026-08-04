package tfchanges

import (
	"strings"
	"testing"
)

func TestNormalizeClassifiesAndFindsFacts(t *testing.T) {
	plan, err := Decode(strings.NewReader(`{"format_version":"1.2","terraform_version":"1.9.0","resource_changes":[{"address":"aws_iam_role.example","mode":"managed","type":"aws_iam_role","name":"example","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["delete","create"],"after_unknown":{"id":true},"after_sensitive":{"secret":true},"after":{"cidr_blocks":["0.0.0.0/0"]}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	records := Normalize(plan, Filter{})
	if len(records) != 1 || records[0].Classification != "replace" || !records[0].Unknown || !records[0].Sensitive {
		t.Fatalf("unexpected records: %#v", records)
	}
	got := map[string]bool{}
	for _, fact := range records[0].Facts {
		got[fact.Code] = true
	}
	for _, code := range []string{"destructive_change", "iam_or_policy_change", "unknown_value", "sensitive_value", "public_network_exposure_candidate"} {
		if !got[code] {
			t.Errorf("missing fact %s", code)
		}
	}
}

func TestDecodeRejectsUnsupportedVersion(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"format_version":"2.0"}`))
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error = %v", err)
	}
}

func TestFilter(t *testing.T) {
	filter, err := CompileFilter("^aws_", "", "", "", "delete")
	if err != nil {
		t.Fatal(err)
	}
	change := resourceChange{Address: "aws_s3_bucket.example"}
	change.Change.Actions = []string{"create"}
	if matches(change, filter) {
		t.Fatal("create should not match delete filter")
	}
}
