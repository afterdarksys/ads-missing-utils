package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const Version = "0.1.0-dev"

const (
	ExitOK      = 0
	ExitFailure = 1
	ExitUsage   = 2
	ExitPartial = 3
	ExitRuntime = 4
	ExitTimeout = 124
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(code int, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var codeErr *Error
	if errors.As(err, &codeErr) {
		return codeErr.Code
	}
	return ExitRuntime
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

// ReorderInterspersed lets commands accept GNU-style flags before or after
// positional arguments while retaining the standard library flag package.
func ReorderInterspersed(args []string, valueFlags map[string]bool) []string {
	flags, positional := make([]string, 0, len(args)), make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") || arg == "--" {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.SplitN(arg, "=", 2)[0]
		if valueFlags[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Field   string `json:"field,omitempty"`
}

type Response[T any] struct {
	Schema      string       `json:"schema"`
	Command     string       `json:"command"`
	Outcome     string       `json:"outcome"`
	Data        T            `json:"data"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}
