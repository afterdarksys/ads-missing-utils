package envsub

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/jwalk"
	"gopkg.in/yaml.v3"
)

const Schema = "missing-utils/envsub/v1"

var placeholder = regexp.MustCompile(`\$\{([^}]*)\}`)
var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Field struct {
	Type     string `yaml:"type" json:"type"`
	Default  any    `yaml:"default" json:"default,omitempty"`
	Values   []any  `yaml:"values" json:"values,omitempty"`
	Secret   bool   `yaml:"secret" json:"secret,omitempty"`
	Optional bool   `yaml:"optional" json:"optional,omitempty"`
}

type Options struct {
	Input        string
	Output       string
	EnvFiles     []string
	Sets         []string
	SchemaPath   string
	StrictSchema bool
	Check        bool
	ListKeys     bool
	Explain      bool
}

type Resolution struct {
	Key    string `json:"key"`
	Source string `json:"source"`
	Value  string `json:"value,omitempty"`
	Secret bool   `json:"secret,omitempty"`
}

type Result struct {
	Schema      string           `json:"schema"`
	Outcome     string           `json:"outcome"`
	Rendered    string           `json:"rendered,omitempty"`
	Keys        []string         `json:"keys,omitempty"`
	Resolutions []Resolution     `json:"resolutions,omitempty"`
	Diagnostics []cli.Diagnostic `json:"diagnostics,omitempty"`
}

func Run(options Options) (Result, error) {
	if options.Input == "" {
		return Result{}, cli.NewError(cli.ExitUsage, "--input is required")
	}
	template, err := os.ReadFile(options.Input)
	if err != nil {
		return Result{}, cli.NewError(cli.ExitRuntime, "reading template: %v", err)
	}
	schema, err := loadSchema(options.SchemaPath)
	if err != nil {
		return Result{}, err
	}
	values, sources, supplied, err := loadValues(options)
	if err != nil {
		return Result{}, err
	}
	if options.StrictSchema && options.SchemaPath != "" {
		for key := range supplied {
			if _, ok := schema[key]; !ok {
				return Result{}, cli.NewError(cli.ExitUsage, "unknown schema key %q", key)
			}
		}
	}
	keys, err := referencedKeys(string(template))
	if err != nil {
		return Result{}, err
	}
	placeholderDefaults, err := defaultsInTemplate(string(template))
	if err != nil {
		return Result{}, err
	}
	sort.Strings(keys)
	result := Result{Schema: Schema, Outcome: "pass", Keys: keys}
	resolved := map[string]string{}
	for _, key := range keys {
		value, source, found := values[key], sources[key], false
		if _, ok := values[key]; ok {
			found = true
		}
		field, declared := schema[key]
		if !declared {
			field = Field{Type: "string"}
		}
		if !found && declared && field.Default != nil {
			value, source, found = stringify(field.Default), "schema_default", true
		}
		if !found {
			if fallback, ok := placeholderDefaults[key]; ok {
				value, source, found = fallback, "placeholder_default", true
			}
		}
		if !found && declared && field.Optional {
			value, source, found = "", "optional", true
		}
		if !found {
			return Result{}, cli.NewError(cli.ExitUsage, "missing required variable %q", key)
		}
		if err := validate(key, value, field); err != nil {
			if field.Secret {
				return Result{}, cli.NewError(cli.ExitUsage, "invalid value for secret key %q: %v", key, err)
			}
			return Result{}, cli.NewError(cli.ExitUsage, "invalid value for %q: %v", key, err)
		}
		resolved[key] = value
		display := value
		if field.Secret {
			display = "***"
		}
		result.Resolutions = append(result.Resolutions, Resolution{Key: key, Source: source, Value: display, Secret: field.Secret})
	}
	if options.ListKeys {
		return result, nil
	}
	rendered, err := render(string(template), resolved, schema)
	if err != nil {
		return Result{}, err
	}
	if options.Explain || options.Check {
		return result, nil
	}
	result.Rendered = rendered
	if options.Output == "" {
		return result, nil
	}
	if err := atomicWrite(options.Output, []byte(rendered)); err != nil {
		return Result{}, err
	}
	return result, nil
}

func loadSchema(path string) (map[string]Field, error) {
	if path == "" {
		return map[string]Field{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, cli.NewError(cli.ExitRuntime, "reading schema: %v", err)
	}
	result := map[string]Field{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, cli.NewError(cli.ExitUsage, "parsing schema: %v", err)
	}
	for key, field := range result {
		if !identifier.MatchString(key) {
			return nil, cli.NewError(cli.ExitUsage, "invalid schema key %q", key)
		}
		if field.Type == "" {
			field.Type = "string"
			result[key] = field
		}
		if err := validateType(field); err != nil {
			return nil, cli.NewError(cli.ExitUsage, "schema key %q: %v", key, err)
		}
	}
	return result, nil
}

func loadValues(options Options) (map[string]string, map[string]string, map[string]bool, error) {
	values, sources, supplied := map[string]string{}, map[string]string{}, map[string]bool{}
	for _, file := range options.EnvFiles {
		entries, err := parseEnvFile(file)
		if err != nil {
			return nil, nil, nil, err
		}
		for key, value := range entries {
			values[key], sources[key], supplied[key] = value, "env_file:"+file, true
		}
	}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key], sources[key] = value, "environment"
		}
	}
	for _, item := range options.Sets {
		key, value, ok := strings.Cut(item, "=")
		if !ok || !identifier.MatchString(key) {
			return nil, nil, nil, cli.NewError(cli.ExitUsage, "--set requires NAME=value")
		}
		values[key], sources[key], supplied[key] = value, "--set", true
	}
	return values, sources, supplied, nil
}

func parseEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, cli.NewError(cli.ExitRuntime, "reading env file %q: %v", path, err)
	}
	result := map[string]string{}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || !identifier.MatchString(strings.TrimSpace(key)) {
			return nil, cli.NewError(cli.ExitUsage, "env file %q line %d is malformed", path, i+1)
		}
		result[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return result, nil
}

func referencedKeys(template string) ([]string, error) {
	seen := map[string]bool{}
	for _, match := range placeholder.FindAllStringSubmatch(template, -1) {
		key, _, err := parsePlaceholder(match[0], match[1])
		if err != nil {
			return nil, err
		}
		seen[key] = true
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	return keys, nil
}

func defaultsInTemplate(template string) (map[string]string, error) {
	defaults := map[string]string{}
	for _, match := range placeholder.FindAllStringSubmatch(template, -1) {
		key, fallback, err := parsePlaceholder(match[0], match[1])
		if err != nil {
			return nil, err
		}
		if fallback != nil {
			defaults[key] = *fallback
		}
	}
	return defaults, nil
}

func parsePlaceholder(full, expr string) (string, *string, error) {
	key, fallback, hasFallback := strings.Cut(expr, ":-")
	if !identifier.MatchString(key) || (strings.Contains(expr, ":") && !hasFallback) {
		return "", nil, cli.NewError(cli.ExitUsage, "unsupported placeholder %q", full)
	}
	if hasFallback {
		return key, &fallback, nil
	}
	return key, nil, nil
}

func render(template string, values map[string]string, schema map[string]Field) (string, error) {
	var renderErr error
	result := placeholder.ReplaceAllStringFunc(template, func(match string) string {
		expr := placeholder.FindStringSubmatch(match)[1]
		key, fallback, hasFallback := strings.Cut(expr, ":-")
		value, ok := values[key]
		if !ok || value == "" {
			if hasFallback {
				value = fallback
			} else {
				renderErr = cli.NewError(cli.ExitUsage, "missing required variable %q", key)
				return match
			}
		}
		if field, ok := schema[key]; ok {
			if err := validate(key, value, field); err != nil {
				renderErr = err
				return match
			}
		}
		return value
	})
	return result, renderErr
}

func validateType(field Field) error {
	switch field.Type {
	case "string", "integer", "number", "boolean", "enum", "url", "duration", "byte-size", "json":
		return nil
	default:
		return fmt.Errorf("unsupported type %q", field.Type)
	}
}

func validate(key, value string, field Field) error {
	if err := validateType(field); err != nil {
		return err
	}
	switch field.Type {
	case "integer":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("must be an integer")
		}
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("must be a number")
		}
	case "boolean":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("must be a boolean")
		}
	case "enum":
		for _, allowed := range field.Values {
			if value == stringify(allowed) {
				return nil
			}
		}
		return fmt.Errorf("must be one of the allowed values")
	case "url":
		parsed, err := url.ParseRequestURI(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("must be an absolute URL")
		}
	case "duration":
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("must be a duration")
		}
	case "byte-size":
		if _, err := jwalk.ParseSize(value); err != nil {
			return fmt.Errorf("must be a byte size")
		}
	case "json":
		if !json.Valid([]byte(value)) {
			return fmt.Errorf("must be valid JSON")
		}
	}
	return nil
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		data, _ := json.Marshal(v)
		if strings.HasPrefix(string(data), "\"") {
			return strings.Trim(string(data), `"`)
		}
		return string(data)
	}
}

func atomicWrite(destination string, data []byte) error {
	dir := filepath.Dir(destination)
	info, err := os.Stat(destination)
	mode := os.FileMode(0o600)
	if err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return cli.NewError(cli.ExitRuntime, "stat output: %v", err)
	}
	tmp, err := os.CreateTemp(dir, ".envsub-")
	if err != nil {
		return cli.NewError(cli.ExitRuntime, "create temporary output: %v", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		return cli.NewError(cli.ExitRuntime, "write output: %v", err)
	}
	if err := os.Rename(tmpName, destination); err != nil {
		return cli.NewError(cli.ExitRuntime, "replace output: %v", err)
	}
	return nil
}
