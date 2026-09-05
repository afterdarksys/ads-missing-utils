package contextsnap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIdentityCollection(t *testing.T) {
	for _, cloud := range []string{"aws", "alicloud"} {
		t.Run(cloud, func(t *testing.T) {
			key := "Account"
			binary := "aws"
			if cloud == "alicloud" {
				key = "AccountId"
				binary = "aliyun"
			}
			runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
				if name != binary || !strings.Contains(strings.Join(args, " "), "--profile work --region test-region") {
					t.Fatalf("unexpected invocation: %s %v", name, args)
				}
				return []byte(`{"` + key + `":"123","Arn":"role/test","AccessKeySecret":"never-export-me"}`), nil
			}
			result := Collect(context.Background(), cloud, "work", "test-region", runner)
			if result.Outcome != "pass" || result.Account != "123" || result.Source != "cli" || result.CollectedAt.IsZero() {
				t.Fatalf("%+v", result)
			}
			raw, _ := json.Marshal(result)
			if strings.Contains(string(raw), "never-export-me") {
				t.Fatal("secret retained")
			}
		})
	}
}
func TestIdentityFailuresArePartial(t *testing.T) {
	runner := func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("secret diagnostic") }
	r := Collect(context.Background(), "aws", "work", "test-region", runner)
	raw, _ := json.Marshal(r)
	if r.Outcome != "partial" || strings.Contains(string(raw), "secret diagnostic") {
		t.Fatalf("%s", raw)
	}
	for _, input := range []string{`{}`, `null`, `{"Account":"123","Arn":"bad\u001b[2J"}`, `{"Account":123,"Arn":"role"}`} {
		r = Normalize([]byte(input), "aws", "work", "test-region", "provided", time.Now())
		if r.Outcome != "partial" {
			t.Fatal(input)
		}
	}
	if Validate("aws", "bad\nprofile", "region") == nil {
		t.Fatal("accepted control character")
	}
}
