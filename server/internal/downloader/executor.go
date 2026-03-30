package downloader

import (
	"fmt"
	"os/exec"
)

// Executor abstracts external process execution.
// Allows mock in tests without spawning real processes.
type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

// DefaultExecutor runs commands via exec.Command.
type DefaultExecutor struct{}

// Run executes the named binary with the given args and returns combined output.
func (e *DefaultExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("exec %q: %w\noutput: %s", name, err, string(out))
	}
	return out, nil
}
