package progress

import (
	"fmt"
	"io"
)

// WriterReporter writes progress events as human-readable lines to w.
// Useful for CLI output and log streaming. Safe for concurrent use.
type WriterReporter struct {
	w io.Writer
}

// NewWriterReporter creates a reporter that writes to w (e.g. os.Stderr).
func NewWriterReporter(w io.Writer) *WriterReporter {
	return &WriterReporter{w: w}
}

// Report formats event as a single line and writes it to the underlying writer.
func (r *WriterReporter) Report(_ string, event Event) {
	switch event.Stage {
	case "metadata":
		fmt.Fprintln(r.w, "  → fetching metadata...")
	case "download":
		fmt.Fprintf(r.w, "  → trying strategy: %s\n", event.Message)
	case "done":
		fmt.Fprintln(r.w, "  ✓ download complete")
	case "error":
		fmt.Fprintf(r.w, "  ✗ error: %s\n", event.Message)
	}
}
