package pll

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}
type Report struct {
	Schema   string         `json:"schema"`
	Path     string         `json:"path"`
	Findings []Finding      `json:"findings"`
	Values   map[string]any `json:"values,omitempty"`
}

func Lint(path string) Report {
	report := Report{Schema: "missing-utils/pll/v1", Path: path, Findings: []Finding{}}
	data, err := os.ReadFile(path)
	if err != nil {
		report.Findings = append(report.Findings, Finding{"PLL_READ_FAILED", "error", err.Error()})
		return report
	}
	if strings.HasPrefix(string(data), "bplist") {
		report.Findings = append(report.Findings, Finding{"PLL_BINARY_UNSUPPORTED", "warning", "binary plist parsing is not supported"})
		return report
	}
	values, duplicates, err := Parse(data)
	if err != nil {
		report.Findings = append(report.Findings, Finding{"PLL_MALFORMED", "error", err.Error()})
		return report
	}
	report.Values = values
	for _, key := range duplicates {
		report.Findings = append(report.Findings, Finding{"PLL_DUPLICATE_KEY", "error", "duplicate dictionary key: " + key})
	}
	program, _ := values["Program"].(string)
	args, _ := values["ProgramArguments"].([]any)
	if program == "" && len(args) > 0 {
		program, _ = args[0].(string)
	}
	if program == "" {
		report.Findings = append(report.Findings, Finding{"PLL_PROGRAM_MISSING_DECLARATION", "warning", "no Program or first ProgramArguments value"})
	} else if strings.HasPrefix(program, "/") {
		if _, err := os.Stat(program); err != nil {
			report.Findings = append(report.Findings, Finding{"PLL_PROGRAM_MISSING", "error", "declared executable does not exist: " + program})
		}
	}
	run, _ := values["RunAtLoad"].(bool)
	_, keep := values["KeepAlive"]
	if run && keep {
		report.Findings = append(report.Findings, Finding{"PLL_PERSISTENT_JOB", "warning", "RunAtLoad and KeepAlive create persistent execution"})
	}
	return report
}

func Parse(data []byte) (map[string]any, []string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil, nil, fmt.Errorf("plist root not found")
		}
		if err != nil {
			return nil, nil, err
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "plist" {
			for {
				token, err = decoder.Token()
				if err != nil {
					return nil, nil, err
				}
				if valueStart, ok := token.(xml.StartElement); ok {
					value, dups, err := decodeValue(decoder, valueStart)
					if err != nil {
						return nil, dups, err
					}
					dict, ok := value.(map[string]any)
					if !ok {
						return nil, dups, fmt.Errorf("plist root value is not a dictionary")
					}
					return dict, dups, nil
				}
			}
		}
	}
}
func decodeValue(decoder *xml.Decoder, start xml.StartElement) (any, []string, error) {
	switch start.Name.Local {
	case "dict":
		result := map[string]any{}
		dups := []string{}
		for {
			token, err := decoder.Token()
			if err != nil {
				return nil, dups, err
			}
			if end, ok := token.(xml.EndElement); ok && end.Name.Local == "dict" {
				return result, dups, nil
			}
			keyStart, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			if keyStart.Name.Local != "key" {
				return nil, dups, fmt.Errorf("expected key, found %s", keyStart.Name.Local)
			}
			var key string
			if err := decoder.DecodeElement(&key, &keyStart); err != nil {
				return nil, dups, err
			}
			for {
				token, err = decoder.Token()
				if err != nil {
					return nil, dups, err
				}
				if valueStart, ok := token.(xml.StartElement); ok {
					value, nested, err := decodeValue(decoder, valueStart)
					dups = append(dups, nested...)
					if err != nil {
						return nil, dups, err
					}
					if _, exists := result[key]; exists {
						dups = append(dups, key)
					}
					result[key] = value
					break
				}
				if end, ok := token.(xml.EndElement); ok && end.Name.Local == "dict" {
					return nil, dups, fmt.Errorf("key %q has no value", key)
				}
			}
		}
	case "array":
		values := []any{}
		dups := []string{}
		for {
			token, err := decoder.Token()
			if err != nil {
				return nil, dups, err
			}
			if end, ok := token.(xml.EndElement); ok && end.Name.Local == "array" {
				return values, dups, nil
			}
			if child, ok := token.(xml.StartElement); ok {
				value, nested, err := decodeValue(decoder, child)
				dups = append(dups, nested...)
				if err != nil {
					return nil, dups, err
				}
				values = append(values, value)
			}
		}
	case "true":
		if err := decoder.Skip(); err != nil {
			return nil, nil, err
		}
		return true, nil, nil
	case "false":
		if err := decoder.Skip(); err != nil {
			return nil, nil, err
		}
		return false, nil, nil
	case "string", "key", "data", "date":
		var value string
		if err := decoder.DecodeElement(&value, &start); err != nil {
			return nil, nil, err
		}
		return value, nil, nil
	case "integer":
		var raw string
		if err := decoder.DecodeElement(&raw, &start); err != nil {
			return nil, nil, err
		}
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		return value, nil, err
	case "real":
		var raw string
		if err := decoder.DecodeElement(&raw, &start); err != nil {
			return nil, nil, err
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		return value, nil, err
	default:
		return nil, nil, fmt.Errorf("unsupported plist element %s", start.Name.Local)
	}
}
