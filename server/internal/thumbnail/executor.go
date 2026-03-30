package thumbnail

import "os/exec"

// Executor abstracts external command execution.
// Allows injection of a mock in tests.
type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

// DefaultExecutor calls real system binaries via os/exec.
type DefaultExecutor struct{}

// Run executes the named binary with the given arguments and returns combined output.
func (DefaultExecutor) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}
