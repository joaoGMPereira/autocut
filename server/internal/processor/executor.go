package processor

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// Executor abstracts external process execution.
// Allows mock in tests without spawning real processes.
// Duplicated from downloader — packages are intentionally independent.
type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

// stderrMaxBytes caps the stderr capture buffer to prevent OOM from verbose
// FFmpeg error output on very long clips.
const stderrMaxBytes = 512 * 1024 // 512 KB

// DefaultExecutor runs commands via exec.Command.
type DefaultExecutor struct{}

// Run executes the named binary with the given args.
// stdout is discarded (prevents unbounded buffer growth for long-running FFmpeg encodes).
// stderr is captured in a bounded 512 KB buffer only on failure.
// Using CombinedOutput() for long FFmpeg operations (e.g. 12-min blur encode at 1080x1920)
// causes the internal bytes.Buffer to grow to hundreds of MB (one line per frame on stderr),
// which triggers an OOM kill of the Go process — appearing as a silent crash with no panic log.
func (e *DefaultExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	// Capture stderr in a bounded 512 KB buffer for error diagnostics; discard stdout.
	// This prevents the pipe-buffer OOM that occurs when FFmpeg prints per-frame
	// progress for multi-minute encodes and Go buffers the entire output in RAM.
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
