package transcript

import (
	"fmt"
	"os/exec"
)

// Executor abstracts running external processes, enabling test injection.
// Kotlin ref: ProcessRunner.execute() — wraps os/exec here for testability.
type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

// DefaultExecutor calls real system binaries via os/exec.
type DefaultExecutor struct{}

// Run executes name with args and returns combined stdout+stderr output.
func (DefaultExecutor) Run(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("exec %q: %w\noutput: %s", name, err, out)
	}
	return out, nil
}
