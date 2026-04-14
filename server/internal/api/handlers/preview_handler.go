package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// PreviewHandler handles pipeline run preview generation requests.
type PreviewHandler struct{}

// NewPreviewHandler creates a PreviewHandler.
func NewPreviewHandler() *PreviewHandler {
	return &PreviewHandler{}
}

// PostPreview accepts a preview generation request.
// The actual ffmpeg composition is not yet implemented; this returns 202 Accepted.
// POST /api/pipeline/runs/{id}/preview
func (h *PreviewHandler) PostPreview(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"run_id":  id,
		"status":  "pending",
		"message": "Preview generation is not yet implemented. The ffmpeg processor will be wired in a future update.",
	})
}
