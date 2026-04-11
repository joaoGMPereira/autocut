package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

// UploadHandler handles YouTube upload requests.
type UploadHandler struct {
	hub          *hub.SSEHub
	uploader     *uploader.YouTubeUploader
	quota        *uploader.QuotaTracker
	auth         *uploader.OAuthManager
	pipelineRepo *database.PipelineRunRepo
	strategy     *uploader.UploadStrategy
	log          *slog.Logger
}

// NewUploadHandler creates an UploadHandler.
func NewUploadHandler(
	h *hub.SSEHub,
	ytUploader *uploader.YouTubeUploader,
	quota *uploader.QuotaTracker,
	auth *uploader.OAuthManager,
	pipelineRepo *database.PipelineRunRepo,
) *UploadHandler {
	return &UploadHandler{
		hub:          h,
		uploader:     ytUploader,
		quota:        quota,
		auth:         auth,
		pipelineRepo: pipelineRepo,
		// FP-014: default to sequential strategy (capacity=1) so uploads don't
		// stomp each other. Callers may override via WithStrategy.
		strategy: uploader.NewUploadStrategy("sequential", 1),
		log:      slog.With("component", "api", "handler", "upload"),
	}
}

// WithStrategy overrides the concurrency strategy used for uploads.
func (h *UploadHandler) WithStrategy(s *uploader.UploadStrategy) *UploadHandler {
	h.strategy = s
	return h
}

type uploadRequest struct {
	FilePath    string `json:"file_path"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Privacy     string `json:"privacy"`
	ChannelID   string `json:"channel_id"`
	SessionID   string `json:"session_id"`
	// FP-013: scheduled publish time (RFC3339). When set, video is uploaded as
	// private and YouTube publishes it automatically at this time.
	PublishAt string `json:"publish_at"`
}

// GetUploads handles GET /api/upload — returns empty list for now.
func (h *UploadHandler) GetUploads(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []interface{}{})
}

// PostUpload handles POST /api/upload.
// TODO: Read upload_strategy + upload_parallel_count from settings and use UploadStrategy for batch uploads
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
	// FP-014: Acquire a concurrency slot before starting the actual YouTube API call.
	// In sequential mode (capacity=1) this ensures uploads run one at a time.
	// In parallel mode this limits simultaneous uploads to the configured count.
	if h.strategy != nil {
		h.strategy.Acquire()
		defer h.strategy.Release()
	}

	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]interface{}{
		"bytes_sent": 0, "total_bytes": 0, "percent": 0,
	}})

	ytReq := uploader.UploadRequest{
		FilePath:    req.FilePath,
		Title:       req.Title,
		Description: req.Description,
		Privacy:     req.Privacy,
	}

	// FP-013: wire publish_at → YouTube Data API scheduled upload.
	// When set, the video is uploaded as "private" and YouTube publishes it
	// at the specified time (status.publishAt in the videos.insert call).
	if req.PublishAt != "" {
		t, err := time.Parse(time.RFC3339, req.PublishAt)
		if err != nil {
			h.log.Warn("upload: invalid publish_at, ignoring schedule",
				"jobID", jobID, "publish_at", req.PublishAt, "err", err)
		} else {
			ytReq.ScheduleAt = &t
			// privacyStatus() in youtube.go already sets "private" when ScheduleAt != nil,
			// and Upload() sets vid.Status.PublishAt to the RFC3339 timestamp.
			h.log.Info("upload scheduled", "jobID", jobID, "publish_at", req.PublishAt)
		}
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
			data := map[string]interface{}{"video_id": prog.VideoID, "success": true}
			h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
			if req.SessionID != "" {
				persistStepOutput(h.pipelineRepo, req.SessionID, "upload", data)
			}
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
