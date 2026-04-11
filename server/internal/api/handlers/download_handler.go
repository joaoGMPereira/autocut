package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/progress"
)

// DownloadHandler handles video download requests and SSE progress streaming.
type DownloadHandler struct {
	hub      *hub.SSEHub
	ytDl     *downloader.YouTubeDownloader
	twDl     *downloader.TwitchDownloader
	reporter progress.Reporter
}

// NewDownloadHandler constructs a DownloadHandler.
// If reporter is nil, a NoopReporter is used.
func NewDownloadHandler(h *hub.SSEHub, ytDl *downloader.YouTubeDownloader, twDl *downloader.TwitchDownloader, reporter progress.Reporter) *DownloadHandler {
	if reporter == nil {
		reporter = progress.NoopReporter{}
	}
	return &DownloadHandler{
		hub:      h,
		ytDl:     ytDl,
		twDl:     twDl,
		reporter: reporter,
	}
}

// downloadRequest is the JSON body for POST /api/download.
type downloadRequest struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	OutputDir string `json:"output_dir"`
}

// videoInfoEvent is the SSE payload published after metadata extraction.
type videoInfoEvent struct {
	JobID        string `json:"job_id"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	DurationSec  int    `json:"duration_sec"`
}

// PostDownload accepts a download job request, assigns a job_id, starts the
// download in the background, and returns 202 Accepted with the job_id.
func (h *DownloadHandler) PostDownload(w http.ResponseWriter, r *http.Request) {
	var req downloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	jobID := newJobID()

	go h.runDownload(context.Background(), jobID, req)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}

// GetDownloadStream streams SSE events for the given job id.
// The job id is extracted from the path value "id".
func (h *DownloadHandler) GetDownloadStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	h.hub.ServeSSE(w, r, jobID)
}

// runDownload performs the download pipeline in a background goroutine:
//  1. ExtractMetadata → publish video_info SSE event
//  2. DownloadWithOptions → publish done SSE event
//  3. On any error → publish error SSE event
func (h *DownloadHandler) runDownload(ctx context.Context, jobID string, req downloadRequest) {
	_ = ctx // reserved for future cancellation support

	publishError := func(msg string) {
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "error",
			Data: map[string]string{"job_id": jobID, "message": msg},
		})
	}

	switch req.Type {
	case "twitch":
		info, err := h.twDl.ExtractMetadata(req.URL)
		if err != nil {
			slog.Error("twitch extract metadata failed", "jobID", jobID, "err", err)
			publishError(err.Error())
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "video_info",
			Data: videoInfoEvent{
				JobID:        jobID,
				Title:        info.Title,
				ThumbnailURL: info.ThumbnailURL,
				DurationSec:  int(info.Duration.Seconds()),
			},
		})
		if _, err := h.twDl.DownloadWithOptions(req.URL, req.OutputDir, ""); err != nil {
			slog.Error("twitch download failed", "jobID", jobID, "err", err)
			publishError(err.Error())
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]string{"job_id": jobID}})

	default: // "youtube" and any unrecognised type fall back to YouTube
		info, err := h.ytDl.ExtractMetadata(req.URL)
		if err != nil {
			slog.Error("youtube extract metadata failed", "jobID", jobID, "err", err)
			publishError(err.Error())
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "video_info",
			Data: videoInfoEvent{
				JobID:        jobID,
				Title:        info.Title,
				ThumbnailURL: info.ThumbnailURL,
				DurationSec:  int(info.Duration.Seconds()),
			},
		})
		if _, err := h.ytDl.DownloadWithOptions(req.URL, req.OutputDir, ""); err != nil {
			slog.Error("youtube download failed", "jobID", jobID, "err", err)
			publishError(err.Error())
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]string{"job_id": jobID}})
	}
}

// newJobID generates a random UUID-like job identifier using crypto/rand.
func newJobID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
