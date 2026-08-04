// Package varmerge composes structured configuration sources with visible
// source provenance. It deliberately does not render templates.
package varmerge

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/envsub"
	"gopkg.in/yaml.v3"
)

const Schema = "missing-utils/varmerge/v1"

type Source struct {
	Name   string
	Values map[string]any
}
type Value struct {
	Value  any    `json:"value"`
	Source string `json:"source"`
	Secret bool   `json:"secret,omitempty"`
}
type Result struct {
	Schema     string              `json:"schema"`
	Outcome    string              `json:"outcome"`
	Values     map[string]any      `json:"values"`
	Provenance map[string]Value    `json:"provenance"`
	Shadowed   map[string][]string `json:"shadowed,omitempty"`
}

// Merge applies sources in order, so a later source has higher precedence.
func Merge(sources []Source, schema map[string]envsub.Field, strict bool) (Result, error) {
	result := Result{Schema: Schema, Outcome: "pass", Values: map[string]any{}, Provenance: map[string]Value{}, Shadowed: map[string][]string{}}
	for _, source := range sources {
		for key, value := range source.Values {
			field, known := schema[key]
			if strict && !known {
				return Result{}, cli.NewError(cli.ExitUsage, "unknown schema key %q from %s", key, source.Name)
			}
			if known {
				if err := validateValue(key, value, field); err != nil {
					return Result{}, err
				}
			}
			if previous, ok := result.Provenance[key]; ok {
				result.Shadowed[key] = append(result.Shadowed[key], previous.Source)
			}
			result.Values[key] = value
			result.Provenance[key] = Value{Value: displayValue(value, field.Secret), Source: source.Name, Secret: field.Secret}
		}
	}
	for key, field := range schema {
		if _, ok := result.Values[key]; !ok && field.Default != nil {
			if err := validateValue(key, field.Default, field); err != nil {
				return Result{}, err
			}
			result.Values[key] = field.Default
			result.Provenance[key] = Value{Value: displayValue(field.Default, field.Secret), Source: "schema_default", Secret: field.Secret}
		}
	}
	for key := range result.Shadowed {
		sort.Strings(result.Shadowed[key])
	}
	return result, nil
}

func JSONSource(name string, input io.Reader) (Source, error) {
	var values map[string]any
	decoder := json.NewDecoder(io.LimitReader(input, 64<<20))
	decoder.UseNumber()
	if err := decoder.Decode(&values); err != nil {
		return Source{}, cli.NewError(cli.ExitUsage, "parse JSON source %s: %v", name, err)
	}
	if values == nil {
		return Source{}, cli.NewError(cli.ExitUsage, "JSON source %s must be an object", name)
	}
	return Source{Name: name, Values: values}, nil
}

func FileSource(path string) (Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Source{}, cli.NewError(cli.ExitRuntime, "read source %q: %v", path, err)
	}
	if strings.HasSuffix(path, ".json") {
		return JSONSource(path, strings.NewReader(string(data)))
	}
	values := map[string]any{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return Source{}, cli.NewError(cli.ExitUsage, "parse YAML source %s: %v", path, err)
	}
	return Source{Name: path, Values: values}, nil
}

func EnvSource(prefix string) Source {
	values := map[string]any{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok && strings.HasPrefix(key, prefix) {
			values[strings.TrimPrefix(key, prefix)] = value
		}
	}
	return Source{Name: "environment:" + prefix, Values: values}
}
func EnvFileSource(path string) (Source, error) {
	values, err := envsub.ParseEnvFile(path)
	if err != nil {
		return Source{}, err
	}
	converted := map[string]any{}
	for key, value := range values {
		converted[key] = value
	}
	return Source{Name: "env_file:" + path, Values: converted}, nil
}
func SetSource(sets []string) (Source, error) {
	values := map[string]any{}
	for _, set := range sets {
		key, value, ok := strings.Cut(set, "=")
		if !ok || key == "" {
			return Source{}, cli.NewError(cli.ExitUsage, "--set requires NAME=value")
		}
		values[key] = value
	}
	return Source{Name: "--set", Values: values}, nil
}

func validateValue(key string, value any, field envsub.Field) error {
	if field.Type == "" || field.Type == "string" {
		return nil
	}
	text := fmt.Sprint(value)
	switch field.Type {
	case "integer":
		if _, err := strconv.ParseInt(text, 10, 64); err != nil {
			return validationError(key, field, "must be an integer")
		}
	case "number":
		if _, err := strconv.ParseFloat(text, 64); err != nil {
			return validationError(key, field, "must be a number")
		}
	case "boolean":
		if _, err := strconv.ParseBool(text); err != nil {
			return validationError(key, field, "must be a boolean")
		}
	case "enum":
		for _, allowed := range field.Values {
			if fmt.Sprint(allowed) == text {
				return nil
			}
		}
		return validationError(key, field, "must be one of the allowed values")
	case "json":
		if _, err := json.Marshal(value); err != nil {
			return validationError(key, field, "must be JSON")
		}
	}
	return nil
}
func validationError(key string, field envsub.Field, message string) error {
	if field.Secret {
		return cli.NewError(cli.ExitUsage, "invalid value for secret key %q: %s", key, message)
	}
	return cli.NewError(cli.ExitUsage, "invalid value for %q: %s", key, message)
}
func displayValue(value any, secret bool) any {
	if secret {
		return "***"
	}
	return value
}
