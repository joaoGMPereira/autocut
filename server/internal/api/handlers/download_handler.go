package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// DownloadHandler handles video download requests (YouTube + Twitch).
type DownloadHandler struct {
	hub          *hub.SSEHub
	ytDl         *downloader.YouTubeDownloader
	twDl         *downloader.TwitchDownloader
	pipelineRepo *database.PipelineRunRepo
	log          *slog.Logger
}

// NewDownloadHandler creates a DownloadHandler.
func NewDownloadHandler(h *hub.SSEHub, ytDl *downloader.YouTubeDownloader, twDl *downloader.TwitchDownloader, pipelineRepo *database.PipelineRunRepo) *DownloadHandler {
	return &DownloadHandler{
		hub:          h,
		ytDl:         ytDl,
		twDl:         twDl,
		pipelineRepo: pipelineRepo,
		log:          slog.With("component", "api", "handler", "download"),
	}
}

type downloadRequest struct {
	URL           string `json:"url"`
	Type          string `json:"type"`       // "youtube" | "twitch"
	OutputDir     string `json:"output_dir"`
	Quality       string `json:"quality"`        // "720p" | "1080p" | "best" | "" (default 1080p)
	WithThumbnail bool   `json:"with_thumbnail"` // download thumbnail alongside video
	WithMetadata  bool   `json:"with_metadata"`  // include title, duration_sec, platform in done event
	SessionID     string `json:"session_id"`
}

// PostDownload handles POST /api/download.
// Returns 202 with job_id; progress is streamed via GET /api/download/{id}/stream.
func (h *DownloadHandler) PostDownload(w http.ResponseWriter, r *http.Request) {
	var req downloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("download: decode body", "err", err)
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "url is required")
		return
	}
	if req.OutputDir == "" {
		req.OutputDir = filepath.Join(os.TempDir(), "autocut-downloads")
		os.MkdirAll(req.OutputDir, 0o755)
	}
	if req.Type == "" {
		req.Type = "youtube"
	}

	jobID := newJobID()
	h.log.Info("download job started", "jobID", jobID, "type", req.Type, "url", req.URL)

	go h.runDownload(jobID, req)

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

// runDownload executes the download in a goroutine and publishes SSE events.
func (h *DownloadHandler) runDownload(jobID string, req downloadRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "starting"}})

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "progress",
		Data: map[string]interface{}{"status": "downloading", "percent": 0},
	})

	switch req.Type {
	case "twitch":
		info, err := h.twDl.DownloadVOD(req.URL, req.OutputDir)
		if err != nil {
			h.log.Error("twitch download failed", "jobID", jobID, "err", err)
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
			return
		}
		data := map[string]interface{}{
			"video_id":  info.VideoID,
			"file_path": info.FilePath,
			"success":   true,
		}
		if req.WithMetadata {
			data["title"] = info.Title
			data["duration_sec"] = info.Duration.Seconds()
			data["platform"] = "twitch"
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
		if req.SessionID != "" {
			persistStepOutput(h.pipelineRepo, req.SessionID, "download", data)
		}

	default: // youtube
		info, err := h.ytDl.DownloadWithOptions(req.URL, req.OutputDir, req.Quality)
		if err != nil {
			h.log.Error("youtube download failed", "jobID", jobID, "err", err)
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
			return
		}
		data := map[string]interface{}{
			"video_id":  info.VideoID,
			"file_path": info.FilePath,
			"success":   true,
		}
		if req.WithThumbnail && info.ThumbnailURL != "" {
			thumbPath := filepath.Join(req.OutputDir, info.VideoID+".thumb.jpg")
			if err := h.ytDl.DownloadThumbnail(req.URL, thumbPath); err != nil {
				h.log.Warn("thumbnail download failed", "jobID", jobID, "err", err)
			} else {
				data["thumbnail_path"] = thumbPath
			}
		}
		if req.WithMetadata {
			data["title"] = info.Title
			data["duration_sec"] = info.Duration.Seconds()
			data["platform"] = "youtube"
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
		if req.SessionID != "" {
			persistStepOutput(h.pipelineRepo, req.SessionID, "download", data)
		}
	}
}

// GetDownloadStream handles GET /api/download/{id}/stream.
func (h *DownloadHandler) GetDownloadStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}
