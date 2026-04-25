// server/internal/api/handlers/logs_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/logsink"
)

// LogsHandler serves the in-app log stream.
type LogsHandler struct {
	sink *logsink.LogSink
	log  *slog.Logger
}

// NewLogsHandler creates a LogsHandler backed by the given LogSink.
func NewLogsHandler(sink *logsink.LogSink) *LogsHandler {
	return &LogsHandler{
		sink: sink,
		log:  slog.With("component", "handler.logs"),
	}
}

// GetHistory returns the current ring buffer as a JSON array.
func (h *LogsHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	entries := h.sink.History()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		h.log.Error("failed to encode log history", "err", err)
	}
}

// GetStream opens an SSE connection that replays the buffer then streams live.
func (h *LogsHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch, cancel := h.sink.Register()
	defer cancel()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case entry, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()

		case <-ticker.C:
			if _, err := fmt.Fprintf(w, "data: {\"ping\":true}\n\n"); err != nil {
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}
