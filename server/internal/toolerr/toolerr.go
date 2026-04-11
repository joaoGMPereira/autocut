// Package toolerr defines a sentinel error type that identifies which external
// tool (ffmpeg, whisper, ollama) caused a failure. Wrapping execution errors
// with ToolErr allows the pipeline service to publish tool-specific SSE events
// so the frontend can surface visual notifications in the dispatcher.
package toolerr

import "fmt"

// ToolErr is an error that carries the identity of the failing tool.
// Use errors.As to unwrap it from any error chain.
type ToolErr struct {
	Tool  string // "ffmpeg" | "whisper" | "ollama"
	Op    string // short operation label, e.g. "cut clip", "transcribe"
	Cause error  // underlying execution error
}

func (e *ToolErr) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s %s: %v", e.Tool, e.Op, e.Cause)
	}
	return fmt.Sprintf("%s %s: unknown error", e.Tool, e.Op)
}

// Unwrap allows errors.As / errors.Is to traverse the chain.
func (e *ToolErr) Unwrap() error { return e.Cause }
