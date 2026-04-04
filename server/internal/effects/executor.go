package effects

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// Executor abstracts external process execution.
// Duplicated per package — packages are intentionally independent (Go design).
type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

// DefaultExecutor runs commands via exec.Command.
type DefaultExecutor struct{}

// Run executes the named binary with args.
// stdout is discarded; stderr is captured only on failure.
// Using CombinedOutput() for long FFmpeg operations causes OOM (hundreds of MB per-frame output).
func (e *DefaultExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		return stderrBuf.Bytes(), fmt.Errorf("exec %q: %w\nstderr: %s", name, err, stderrBuf.String())
	}
	return nil, nil
}
