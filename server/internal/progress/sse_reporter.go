package progress

import "github.com/joaoGMPereira/autocut/server/internal/hub"

// SSEReporter delivers progress events to SSE clients via hub.SSEHub.
// It is safe for concurrent use — hub.Publish is non-blocking.
type SSEReporter struct {
	hub *hub.SSEHub
}

// NewSSEReporter wraps h as a Reporter.
func NewSSEReporter(h *hub.SSEHub) *SSEReporter {
	return &SSEReporter{hub: h}
}

// Report converts event to an SSEEvent and publishes it to all listeners for jobID.
// Non-blocking: slow or disconnected listeners are skipped by the hub.
func (r *SSEReporter) Report(jobID string, event Event) {
	r.hub.Publish(jobID, hub.SSEEvent{
		Type: event.Stage,
		Data: event,
	})
}
