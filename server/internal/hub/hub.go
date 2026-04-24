// Package hub provides a per-job SSE (Server-Sent Events) fan-out hub.
// It lives in its own package to avoid import cycles between api and handlers.
package hub

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SSEEvent is the envelope for all server-sent events.
type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// SSEHub manages per-job SSE listener channels.
// Multiple listeners per job are supported (e.g. two browser tabs).
//
// Replay buffer: events published before any listener registers are buffered
// (up to maxHistory per job) and replayed to the first listener that joins.
// This prevents the race where a fast-completing job (e.g. manual-install
// validators that return an error immediately) publishes its terminal event
// before the browser EventSource has finished opening the stream.
type SSEHub struct {
	mu      sync.Mutex
	clients map[string][]chan SSEEvent
	history map[string][]SSEEvent
	log     *slog.Logger
}

const maxHistory = 128

// New creates an initialised SSEHub.
func New() *SSEHub {
	return &SSEHub{
		clients: make(map[string][]chan SSEEvent),
		history: make(map[string][]SSEEvent),
		log:     slog.With("component", "hub.sse"),
	}
}

// Register adds a new listener channel for jobID and returns:
//   - the receive-only channel the caller reads events from
//   - a cancel function that must be called when the listener disconnects
//
// The channel is buffered (128) so a slow consumer does not block Publish.
// Any events already in the replay buffer are drained into the channel
// immediately so late-joining listeners receive the full history.
func (h *SSEHub) Register(jobID string) (<-chan SSEEvent, func()) {
	ch := make(chan SSEEvent, 128)

	h.mu.Lock()
	h.clients[jobID] = append(h.clients[jobID], ch)
	// Replay buffered events so a listener that arrives after Publish still
	// receives all prior events (including terminal 'done'/'error').
	for _, event := range h.history[jobID] {
		select {
		case ch <- event:
		default:
		}
	}
	h.mu.Unlock()

	h.log.Debug("SSE listener registered", "jobID", jobID)

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		listeners := h.clients[jobID]
		for i, c := range listeners {
			if c == ch {
				h.clients[jobID] = append(listeners[:i], listeners[i+1:]...)
				close(ch)
				break
			}
		}
		if len(h.clients[jobID]) == 0 {
			delete(h.clients, jobID)
			delete(h.history, jobID)
		}
		h.log.Debug("SSE listener cancelled", "jobID", jobID)
	}

	return ch, cancel
}

// Publish sends event to every registered listener for jobID and appends it
// to the replay buffer so listeners that join later receive the full history.
// Non-blocking: slow/full channels are skipped.
func (h *SSEHub) Publish(jobID string, event SSEEvent) {
	h.mu.Lock()
	if len(h.history[jobID]) < maxHistory {
		h.history[jobID] = append(h.history[jobID], event)
	}
	listeners := make([]chan SSEEvent, len(h.clients[jobID]))
	copy(listeners, h.clients[jobID])
	h.mu.Unlock()

	for _, ch := range listeners {
		select {
		case ch <- event:
		default:
			h.log.Warn("SSE channel full, dropping event", "jobID", jobID, "type", event.Type)
		}
	}
}

// ServeSSE registers as a listener for jobID and streams events to w until
// the client disconnects or the request context is cancelled.
//
// Headers set:
//
//	Content-Type: text/event-stream
//	Cache-Control: no-cache
//	X-Accel-Buffering: no
func (h *SSEHub) ServeSSE(w http.ResponseWriter, r *http.Request, jobID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch, cancel := h.Register(jobID)
	defer cancel()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, open := <-ch:
			if !open {
				return
			}
			if err := writeSSEEvent(w, event); err != nil {
				h.log.Error("SSE write failed", "jobID", jobID, "err", err)
				return
			}
			flusher.Flush()

		case <-ticker.C:
			if _, err := fmt.Fprintf(w, "data: {\"type\":\"ping\"}\n\n"); err != nil {
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// writeSSEEvent serialises event as "data: {json}\n\n".
func writeSSEEvent(w http.ResponseWriter, event SSEEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal SSE event: %w", err)
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}
