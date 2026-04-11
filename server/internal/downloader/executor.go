package downloader

import "os/exec"

// Executor runs an external process and captures its combined stdout/stderr output.
type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

// OSExecutor is the production implementation of Executor.
type OSExecutor struct{}

func (e OSExecutor) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}
