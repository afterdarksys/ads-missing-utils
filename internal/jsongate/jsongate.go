// Package jsongate converts JSON findings into an explicit automation decision.
package jsongate

import (
	"bufio"
	"encoding/json"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"sort"
	"strings"
)

const Schema = "missing-utils/jsongate/v1"

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message,omitempty"`
}
type Policy struct {
	DenyAt     string   `yaml:"deny_at"`
	AllowCodes []string `yaml:"allow_codes"`
	DenyCodes  []string `yaml:"deny_codes"`
	ApprovalAt string   `yaml:"approval_at"`
}
type Result struct {
	Schema   string    `json:"schema"`
	Decision string    `json:"decision"`
	Findings []Finding `json:"findings"`
	Reasons  []Finding `json:"reasons"`
}

func LoadPolicy(path string) (Policy, error) {
	if path == "" {
		return Policy{DenyAt: "high"}, nil
	}
	data, err := osRead(path)
	if err != nil {
		return Policy{}, cli.NewError(cli.ExitRuntime, "read policy: %v", err)
	}
	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return Policy{}, cli.NewError(cli.ExitUsage, "parse policy: %v", err)
	}
	if policy.DenyAt == "" {
		policy.DenyAt = "high"
	}
	if !validSeverity(policy.DenyAt) || (policy.ApprovalAt != "" && !validSeverity(policy.ApprovalAt)) {
		return Policy{}, cli.NewError(cli.ExitUsage, "policy has invalid severity")
	}
	return policy, nil
}

var osRead = func(path string) ([]byte, error) { return os.ReadFile(path) }

func Decode(input io.Reader) ([]Finding, error) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	findings := []Finding{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var finding Finding
		if err := json.Unmarshal([]byte(line), &finding); err != nil {
			return nil, cli.NewError(cli.ExitUsage, "invalid finding JSON: %v", err)
		}
		if finding.Code == "" || !validSeverity(finding.Severity) {
			return nil, cli.NewError(cli.ExitUsage, "finding requires code and severity (info, low, medium, high, critical)")
		}
		findings = append(findings, finding)
	}
	if err := scanner.Err(); err != nil {
		return nil, cli.NewError(cli.ExitRuntime, "read findings: %v", err)
	}
	return findings, nil
}
func Evaluate(findings []Finding, policy Policy) Result {
	allow, deny := set(policy.AllowCodes), set(policy.DenyCodes)
	result := Result{Schema: Schema, Decision: "pass", Findings: findings}
	for _, finding := range findings {
		if deny[finding.Code] || (!allow[finding.Code] && rank(finding.Severity) >= rank(policy.DenyAt)) {
			result.Decision = "deny"
			result.Reasons = append(result.Reasons, finding)
			continue
		}
		if result.Decision == "pass" && policy.ApprovalAt != "" && !allow[finding.Code] && rank(finding.Severity) >= rank(policy.ApprovalAt) {
			result.Decision = "approval_required"
			result.Reasons = append(result.Reasons, finding)
		}
	}
	sort.Slice(result.Reasons, func(i, j int) bool { return result.Reasons[i].Code < result.Reasons[j].Code })
	return result
}
func set(values []string) map[string]bool {
	r := map[string]bool{}
	for _, v := range values {
		r[v] = true
	}
	return r
}
func validSeverity(value string) bool { return rank(value) >= 0 }
func rank(value string) int {
	switch value {
	case "info":
		return 0
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	}
	return -1
}
