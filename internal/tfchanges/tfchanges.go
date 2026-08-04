// Package tfchanges normalizes Terraform and OpenTofu JSON plans into stable
// resource-change records suitable for automation.
package tfchanges

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
)

const Schema = "missing-utils/tfchanges/v1"
const maxInputBytes = 64 << 20

type Plan struct {
	FormatVersion    string           `json:"format_version"`
	TerraformVersion string           `json:"terraform_version"`
	ResourceChanges  []resourceChange `json:"resource_changes"`
}

type resourceChange struct {
	Address       string `json:"address"`
	ModuleAddress string `json:"module_address"`
	Mode          string `json:"mode"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	ProviderName  string `json:"provider_name"`
	Change        struct {
		Actions         []string `json:"actions"`
		Before          any      `json:"before"`
		After           any      `json:"after"`
		AfterUnknown    any      `json:"after_unknown"`
		BeforeSensitive any      `json:"before_sensitive"`
		AfterSensitive  any      `json:"after_sensitive"`
		ReplacePaths    any      `json:"replace_paths"`
	} `json:"change"`
}

type Fact struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

type Record struct {
	Schema           string   `json:"schema"`
	Status           string   `json:"status"`
	Address          string   `json:"address"`
	ModuleAddress    string   `json:"module_address,omitempty"`
	Mode             string   `json:"mode"`
	Provider         string   `json:"provider,omitempty"`
	ResourceType     string   `json:"resource_type"`
	Name             string   `json:"name"`
	Actions          []string `json:"actions"`
	Classification   string   `json:"classification"`
	Replacement      bool     `json:"replacement"`
	Unknown          bool     `json:"unknown"`
	Sensitive        bool     `json:"sensitive"`
	ReplacementPaths any      `json:"replacement_paths,omitempty"`
	Facts            []Fact   `json:"facts,omitempty"`
}

type Summary struct {
	Schema           string         `json:"schema"`
	Status           string         `json:"status"`
	TerraformVersion string         `json:"terraform_version,omitempty"`
	Total            int            `json:"total"`
	Classifications  map[string]int `json:"classifications"`
	FactCounts       map[string]int `json:"fact_counts"`
}

type Filter struct {
	Address, Module, Provider, Type *regexp.Regexp
	Actions                         map[string]bool
}

// Decode validates and decodes a plan. The limit prevents unbounded stdin
// allocation in CI and hook environments.
func Decode(input io.Reader) (Plan, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxInputBytes+1))
	if err != nil {
		return Plan{}, cli.NewError(cli.ExitRuntime, "read plan: %v", err)
	}
	if len(data) > maxInputBytes {
		return Plan{}, cli.NewError(cli.ExitUsage, "plan exceeds 64 MiB input limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var plan Plan
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, cli.NewError(cli.ExitUsage, "invalid plan JSON: %v", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return Plan{}, cli.NewError(cli.ExitUsage, "plan JSON must contain one document")
	}
	if plan.FormatVersion == "" {
		return Plan{}, cli.NewError(cli.ExitUsage, "plan is missing format_version")
	}
	if !strings.HasPrefix(plan.FormatVersion, "1.") {
		return Plan{}, cli.NewError(cli.ExitUsage, "unsupported plan format_version %q", plan.FormatVersion)
	}
	return plan, nil
}

func Normalize(plan Plan, filter Filter) []Record {
	records := make([]Record, 0, len(plan.ResourceChanges))
	for _, change := range plan.ResourceChanges {
		if !matches(change, filter) {
			continue
		}
		actions := append([]string(nil), change.Change.Actions...)
		replacement := contains(actions, "create") && contains(actions, "delete")
		record := Record{Schema: Schema, Status: "ok", Address: change.Address, ModuleAddress: change.ModuleAddress, Mode: change.Mode, Provider: change.ProviderName, ResourceType: change.Type, Name: change.Name, Actions: actions, Classification: classify(actions), Replacement: replacement, Unknown: hasTrue(change.Change.AfterUnknown), Sensitive: hasTrue(change.Change.BeforeSensitive) || hasTrue(change.Change.AfterSensitive), ReplacementPaths: change.Change.ReplacePaths}
		record.Facts = facts(record, change.Change.After)
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Address < records[j].Address })
	return records
}

func Summarize(plan Plan, records []Record) Summary {
	summary := Summary{Schema: Schema, Status: "ok", TerraformVersion: plan.TerraformVersion, Total: len(records), Classifications: map[string]int{}, FactCounts: map[string]int{}}
	for _, record := range records {
		summary.Classifications[record.Classification]++
		for _, fact := range record.Facts {
			summary.FactCounts[fact.Code]++
		}
	}
	return summary
}

func matches(change resourceChange, filter Filter) bool {
	if filter.Address != nil && !filter.Address.MatchString(change.Address) {
		return false
	}
	if filter.Module != nil && !filter.Module.MatchString(change.ModuleAddress) {
		return false
	}
	if filter.Provider != nil && !filter.Provider.MatchString(change.ProviderName) {
		return false
	}
	if filter.Type != nil && !filter.Type.MatchString(change.Type) {
		return false
	}
	if len(filter.Actions) != 0 {
		matched := false
		for _, action := range change.Change.Actions {
			if filter.Actions[action] {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func classify(actions []string) string {
	switch {
	case len(actions) == 0:
		return "unknown"
	case len(actions) == 1 && actions[0] == "no-op":
		return "no-op"
	case len(actions) == 1 && actions[0] == "create":
		return "create"
	case len(actions) == 1 && actions[0] == "read":
		return "read"
	case len(actions) == 1 && actions[0] == "update":
		return "update"
	case len(actions) == 1 && actions[0] == "delete":
		return "delete"
	case contains(actions, "create") && contains(actions, "delete"):
		return "replace"
	default:
		return "unknown"
	}
}

func facts(record Record, after any) []Fact {
	facts := []Fact{}
	if record.Classification == "delete" || record.Classification == "replace" {
		facts = append(facts, Fact{"destructive_change", "lifecycle", "resource is deleted or replaced"})
	}
	typeName := strings.ToLower(record.ResourceType)
	if strings.Contains(typeName, "iam") || strings.Contains(typeName, "policy") || strings.Contains(typeName, "role") {
		facts = append(facts, Fact{"iam_or_policy_change", "identity", "resource type suggests an identity or policy change"})
	}
	if strings.Contains(typeName, "database") || strings.Contains(typeName, "db_") || strings.Contains(typeName, "rds") {
		facts = append(facts, Fact{"database_change", "data", "resource type suggests a database change"})
	}
	if record.Sensitive {
		facts = append(facts, Fact{"sensitive_value", "sensitivity", "plan marks one or more values sensitive"})
	}
	if record.Unknown {
		facts = append(facts, Fact{"unknown_value", "uncertainty", "plan contains values unknown until apply"})
	}
	if publicExposure(after) {
		facts = append(facts, Fact{"public_network_exposure_candidate", "network", "after-state contains a public address or unrestricted CIDR"})
	}
	return facts
}

func publicExposure(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			key = strings.ToLower(key)
			if (key == "cidr_block" || key == "cidr_blocks" || key == "ipv6_cidr_block" || key == "ipv6_cidr_blocks") && containsPublicCIDR(child) {
				return true
			}
			if (key == "public_ip" || key == "associate_public_ip_address") && child == true {
				return true
			}
			if publicExposure(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if publicExposure(child) {
				return true
			}
		}
	}
	return false
}

func containsPublicCIDR(value any) bool {
	switch v := value.(type) {
	case string:
		return v == "0.0.0.0/0" || v == "::/0"
	case []any:
		for _, item := range v {
			if containsPublicCIDR(item) {
				return true
			}
		}
	}
	return false
}

func hasTrue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case map[string]any:
		for _, child := range v {
			if hasTrue(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if hasTrue(child) {
				return true
			}
		}
	}
	return false
}
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func CompileFilter(address, module, provider, resourceType, actions string) (Filter, error) {
	compile := func(name, value string) (*regexp.Regexp, error) {
		if value == "" {
			return nil, nil
		}
		r, err := regexp.Compile(value)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", name, err)
		}
		return r, nil
	}
	filter := Filter{}
	var err error
	if filter.Address, err = compile("address", address); err != nil {
		return Filter{}, err
	}
	if filter.Module, err = compile("module", module); err != nil {
		return Filter{}, err
	}
	if filter.Provider, err = compile("provider", provider); err != nil {
		return Filter{}, err
	}
	if filter.Type, err = compile("type", resourceType); err != nil {
		return Filter{}, err
	}
	if actions != "" {
		filter.Actions = map[string]bool{}
		for _, action := range strings.Split(actions, ",") {
			filter.Actions[strings.TrimSpace(action)] = true
		}
	}
	return filter, nil
}
