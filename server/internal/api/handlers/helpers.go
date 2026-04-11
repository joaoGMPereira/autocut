package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// writeJSON serialises v as JSON with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode failed", "err", err)
	}
}

// writeError sends a structured JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   message,
		"message": message,
		"code":    code,
	})
}

// writeToolError sends a structured JSON error response that identifies the
// failing tool (ffmpeg, whisper, ollama). The extra fields allow the frontend
// to surface a visual notification in the dispatcher.
func writeToolError(w http.ResponseWriter, tool, op string, cause error) {
	msg := fmt.Sprintf("%s %s failed", tool, op)
	if cause != nil {
		msg = fmt.Sprintf("%s %s: %v", tool, op, cause)
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"error":   msg,
		"message": msg,
		"code":    tool + "_error",
		"tool":    tool,
		"op":      op,
	})
}
