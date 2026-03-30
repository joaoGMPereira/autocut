package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// DownloadHandler handles video download requests (YouTube + Twitch).
type DownloadHandler struct {
	hub  *hub.SSEHub
	ytDl *downloader.YouTubeDownloader
	twDl *downloader.TwitchDownloader
	log  *slog.Logger
}

// NewDownloadHandler creates a DownloadHandler.
func NewDownloadHandler(h *hub.SSEHub, ytDl *downloader.YouTubeDownloader, twDl *downloader.TwitchDownloader) *DownloadHandler {
	return &DownloadHandler{
		hub:  h,
		ytDl: ytDl,
		twDl: twDl,
		log:  slog.With("component", "api", "handler", "download"),
	}
}

type downloadRequest struct {
	URL       string `json:"url"`
	Type      string `json:"type"`       // "youtube" | "twitch"
	OutputDir string `json:"output_dir"`
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
		writeError(w, http.StatusBadRequest, "missing_field", "output_dir is required")
		return
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
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "done",
			Data: map[string]string{"video_id": info.VideoID, "file_path": info.FilePath},
		})

	default: // youtube
		info, err := h.ytDl.Download(req.URL, req.OutputDir)
		if err != nil {
			h.log.Error("youtube download failed", "jobID", jobID, "err", err)
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "done",
			Data: map[string]string{"video_id": info.VideoID, "file_path": info.FilePath},
		})
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
