package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

// UploadHandler handles YouTube upload requests.
type UploadHandler struct {
	hub      *hub.SSEHub
	uploader *uploader.YouTubeUploader
	quota    *uploader.QuotaTracker
	auth     *uploader.OAuthManager
	log      *slog.Logger
}

// NewUploadHandler creates an UploadHandler.
func NewUploadHandler(
	h *hub.SSEHub,
	ytUploader *uploader.YouTubeUploader,
	quota *uploader.QuotaTracker,
	auth *uploader.OAuthManager,
) *UploadHandler {
	return &UploadHandler{
		hub:      h,
		uploader: ytUploader,
		quota:    quota,
		auth:     auth,
		log:      slog.With("component", "api", "handler", "upload"),
	}
}

type uploadRequest struct {
	FilePath    string `json:"file_path"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Privacy     string `json:"privacy"`
	ChannelID   string `json:"channel_id"`
}

// GetUploads handles GET /api/upload — returns empty list for now.
func (h *UploadHandler) GetUploads(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []interface{}{})
}

// PostUpload handles POST /api/upload.
func (h *UploadHandler) PostUpload(w http.ResponseWriter, r *http.Request) {
	if h.uploader == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "YouTube uploader not configured — complete OAuth setup first")
		return
	}

	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.FilePath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "file_path is required")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "title is required")
		return
	}
	if req.Privacy == "" {
		req.Privacy = "private"
	}

	jobID := newJobID()
	h.log.Info("upload job started", "jobID", jobID, "file", req.FilePath)
	go h.runUpload(r.Context(), jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *UploadHandler) runUpload(ctx context.Context, jobID string, req uploadRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]interface{}{
		"bytes_sent": 0, "total_bytes": 0, "percent": 0,
	}})

	ytReq := uploader.UploadRequest{
		FilePath:    req.FilePath,
		Title:       req.Title,
		Description: req.Description,
		Privacy:     req.Privacy,
	}

	ch, err := h.uploader.Upload(ctx, ytReq)
	if err != nil {
		h.log.Error("upload start failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
		return
	}

	for prog := range ch {
		if prog.Err != nil {
			h.log.Error("upload failed", "jobID", jobID, "err", prog.Err)
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": prog.Err.Error()}})
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "progress",
			Data: map[string]interface{}{
				"bytes_sent":  prog.BytesSent,
				"total_bytes": prog.TotalBytes,
				"percent":     prog.Percent,
			},
		})
		if prog.Done {
			h.hub.Publish(jobID, hub.SSEEvent{
				Type: "done",
				Data: map[string]string{"video_id": prog.VideoID},
			})
			return
		}
	}
}

// GetUploadStream handles GET /api/upload/{id}/stream.
func (h *UploadHandler) GetUploadStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}
