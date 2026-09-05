// Package collectout bounds output from explicitly selected read-only CLIs.
package collectout

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

const Limit = 4 << 20

type Runner func(context.Context, string, ...string) ([]byte, error)
type bounded struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (w *bounded) Write(p []byte) (int, error) {
	if len(p) > Limit-w.buffer.Len() {
		w.exceeded = true
		return 0, fmt.Errorf("collector output limit exceeded")
	}
	return w.buffer.Write(p)
}
func Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	var output bounded
	command.Stdout = &output
	command.Stderr = io.Discard
	command.WaitDelay = time.Second
	err := command.Run()
	if output.exceeded {
		return nil, fmt.Errorf("collector output limit exceeded")
	}
	if err != nil {
		return nil, fmt.Errorf("collector failed or timed out")
	}
	return output.buffer.Bytes(), nil
}
