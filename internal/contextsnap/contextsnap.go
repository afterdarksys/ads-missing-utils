// Package contextsnap normalizes caller identity without retaining credentials.
package contextsnap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/collectout"
)

const Schema = "missing-utils/contextsnap/v1"

type Result struct {
	Schema           string    `json:"schema"`
	Source           string    `json:"source"`
	Cloud            string    `json:"cloud"`
	Profile          string    `json:"profile"`
	Region           string    `json:"region"`
	CollectedAt      time.Time `json:"collected_at"`
	CollectorVersion string    `json:"collector_version"`
	Account          string    `json:"account"`
	Principal        string    `json:"principal"`
	Outcome          string    `json:"outcome"`
	Diagnostics      []string  `json:"diagnostics"`
}

func Validate(cloud, profile, region string) error {
	if cloud != "aws" && cloud != "alicloud" {
		return fmt.Errorf("--cloud must be aws or alicloud")
	}
	for _, s := range []string{profile, region} {
		if !Safe(s) {
			return fmt.Errorf("--profile and --region must be nonblank, single-line values")
		}
	}
	return nil
}
func Safe(s string) bool {
	return strings.TrimSpace(s) != "" && len(s) <= 2048 && !strings.ContainsFunc(s, unicode.IsControl)
}
func Normalize(data []byte, cloud, profile, region, source string, observed time.Time) Result {
	r := Result{Schema: Schema, Source: source, Cloud: cloud, Profile: profile, Region: region, CollectedAt: observed.UTC(), CollectorVersion: cli.Version, Outcome: "partial", Diagnostics: []string{}}
	var raw struct {
		Account   string `json:"Account"`
		AccountID string `json:"AccountId"`
		Arn       string `json:"Arn"`
	}
	if len(data) > collectout.Limit || json.Unmarshal(data, &raw) != nil {
		r.Diagnostics = append(r.Diagnostics, "identity response is invalid or exceeds the input limit")
		return r
	}
	r.Account = raw.Account
	if cloud == "alicloud" {
		r.Account = raw.AccountID
	}
	r.Principal = raw.Arn
	if !Safe(r.Account) || !Safe(r.Principal) {
		r.Account = ""
		r.Principal = ""
		r.Diagnostics = append(r.Diagnostics, "identity response lacks a usable account or principal")
		return r
	}
	if observed.IsZero() {
		r.Diagnostics = append(r.Diagnostics, "observation time is unavailable")
		return r
	}
	r.Outcome = "pass"
	return r
}
func Collect(ctx context.Context, cloud, profile, region string, run collectout.Runner) Result {
	started := time.Now().UTC()
	var data []byte
	var err error
	if cloud == "aws" {
		data, err = run(ctx, "aws", "sts", "get-caller-identity", "--profile", profile, "--region", region, "--output", "json", "--no-cli-pager", "--no-cli-auto-prompt")
	} else {
		data, err = run(ctx, "aliyun", "sts", "GetCallerIdentity", "--profile", profile, "--region", region)
	}
	if err != nil {
		return Result{Schema: Schema, Source: "cli", Cloud: cloud, Profile: profile, Region: region, CollectedAt: started, CollectorVersion: cli.Version, Outcome: "partial", Diagnostics: []string{"caller identity collection failed or timed out; check profile access and CLI availability"}}
	}
	return Normalize(data, cloud, profile, region, "cli", started)
}
