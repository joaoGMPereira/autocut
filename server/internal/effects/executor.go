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

// stderrMaxBytes caps the stderr capture buffer to prevent OOM from verbose
// FFmpeg error output on very long clips.
const stderrMaxBytes = 512 * 1024 // 512 KB

// DefaultExecutor runs commands via exec.Command.
type DefaultExecutor struct{}

// Run executes the named binary with args.
// stdout is discarded; stderr is captured in a bounded 512 KB buffer only on failure.
// Using CombinedOutput() for long FFmpeg operations causes OOM (hundreds of MB per-frame output).
func (e *DefaultExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.Discard
	// LimitWriter caps stderr at 512 KB to prevent unbounded buffer growth
	// when an FFmpeg invocation unexpectedly emits many error lines.
	cmd.Stderr = &limitedWriter{buf: &stderrBuf, limit: stderrMaxBytes}
	if err := cmd.Run(); err != nil {
		return stderrBuf.Bytes(), fmt.Errorf("exec %q: %w\nstderr: %s", name, err, stderrBuf.String())
	}
	return nil, nil
}

// limitedWriter wraps a *bytes.Buffer and stops writing after limit bytes,
// preventing unbounded memory growth from verbose FFmpeg stderr output.
type limitedWriter struct {
	buf     *bytes.Buffer
	limit   int
	written int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.written
	if remaining <= 0 {
		return len(p), nil // silently discard once cap reached
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err := w.buf.Write(p)
	w.written += n
	return n, err
}
