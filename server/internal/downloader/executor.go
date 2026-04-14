package downloader

import (
	"bufio"
	"context"
	"os"
	"os/exec"
)

// Executor runs an external process and captures its combined stdout/stderr output.
type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

// OSExecutor is the production implementation of Executor.
type OSExecutor struct{}

func (e OSExecutor) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// StreamingExecutor runs a command and calls onLine for each line of combined output.
// Returns an error if the command exits non-zero or ctx is cancelled.
type StreamingExecutor interface {
	RunWithContext(ctx context.Context, name string, args []string, onLine func(string)) error
}

// OSStreamingExecutor is the production implementation of StreamingExecutor.
type OSStreamingExecutor struct{}

func (e OSStreamingExecutor) RunWithContext(ctx context.Context, name string, args []string, onLine func(string)) error {
	cmd := exec.CommandContext(ctx, name, args...)
	pr, pw, err := newPipeWithMergedStderr(cmd)
	if err != nil {
		return err
	}
	defer pr.Close()
	if err := cmd.Start(); err != nil {
		pw.Close()
		return err
	}
	pw.Close() // close our write end so scanner gets EOF when cmd exits
	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
	return cmd.Wait()
}

// newPipeWithMergedStderr wires cmd.Stdout and cmd.Stderr both through the
// write end of an OS pipe. The caller reads from the returned read-end.
func newPipeWithMergedStderr(cmd *exec.Cmd) (*os.File, *os.File, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	return pr, pw, nil
}
