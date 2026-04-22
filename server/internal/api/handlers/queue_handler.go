package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// QueueHandler serves the /api/queue endpoints.
type QueueHandler struct {
	repo          *database.UploadRepo
	dataDir       string
	triggerUpload func() // optional: triggers the upload worker immediately
	log           *slog.Logger
}

// NewQueueHandler constructs a QueueHandler.
func NewQueueHandler(repo *database.UploadRepo, dataDir string) *QueueHandler {
	return &QueueHandler{repo: repo, dataDir: dataDir, log: slog.With("handler", "queue")}
}

// SetUploadTrigger sets a function that immediately runs the upload worker.
func (h *QueueHandler) SetUploadTrigger(fn func()) {
	h.triggerUpload = fn
}

// queueItemJSON is the JSON shape returned by GET /api/queue.
type queueItemJSON struct {
	ID            int64   `json:"id"`
	ClipID        int64   `json:"clip_id"`
	ChannelID     int64   `json:"channel_id"`
	Status        string  `json:"status"`
	VideoPath     string  `json:"video_path"`
	QueueOrder    int     `json:"queue_order"`
	PublishAt     *string `json:"publish_at,omitempty"`
	YoutubeID     string  `json:"youtube_id"`
	YoutubeURL    string  `json:"youtube_url"`
	Error         string  `json:"error"`
	CreatedAt     int64   `json:"created_at"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Tags          string  `json:"tags"` // comma-separated from pipeline_clips
	ThumbnailPath string  `json:"thumbnail_path"`
	VideoType     string  `json:"video_type"`
	MetadataJSON  string  `json:"metadata_json"`
}

// GetQueue handles GET /api/queue
func (h *QueueHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListQueueWithTitle(r.Context())
	if err != nil {
		h.log.Error("list queue", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := make([]queueItemJSON, 0, len(items))
	for _, q := range items {
		j := queueItemJSON{
			ID:            q.ID,
			ClipID:        q.ClipID,
			ChannelID:     q.ChannelID,
			Status:        q.Status,
			VideoPath:     q.VideoPath,
			QueueOrder:    q.QueueOrder,
			YoutubeID:     q.YoutubeID,
			YoutubeURL:    q.YoutubeURL,
			Error:         q.Error,
			CreatedAt:     q.CreatedAt,
			Title:         q.Title,
			Description:   q.Description,
			Tags:          q.Tags,
			ThumbnailPath: q.ThumbnailPath,
			VideoType:     q.VideoType,
			MetadataJSON:  q.MetadataJSON,
		}
		if q.PublishAt.Valid {
			j.PublishAt = &q.PublishAt.String
		}
		out = append(out, j)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// DeleteQueue handles DELETE /api/queue/{id}
func (h *QueueHandler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.log.Error("delete queue item", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostRetry handles POST /api/queue/{id}/retry
func (h *QueueHandler) PostRetry(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateStatus(r.Context(), id, "queued"); err != nil {
		h.log.Error("retry queue item", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostSchedule handles POST /api/queue/{id}/schedule
// Body: {"publish_at":"2026-04-22T10:00:00Z"}
func (h *QueueHandler) PostSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		PublishAt string `json:"publish_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PublishAt == "" {
		http.Error(w, "publish_at required", http.StatusBadRequest)
		return
	}
	u, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	configJSON := u.UploadConfigJSON
	if configJSON == "" {
		configJSON = "{}"
	}
	if err := h.repo.SetSchedule(r.Context(), id, body.PublishAt, configJSON); err != nil {
		h.log.Error("set schedule", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostBulkSchedule handles POST /api/queue/bulk-schedule
// Body: {"start_at":"2026-04-22T10:00:00Z","interval_minutes":1440}
func (h *QueueHandler) PostBulkSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StartAt         string `json:"start_at"`
		IntervalMinutes int    `json:"interval_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.StartAt == "" {
		http.Error(w, "start_at required", http.StatusBadRequest)
		return
	}
	if body.IntervalMinutes <= 0 {
		body.IntervalMinutes = 1440
	}
	t, err := time.Parse(time.RFC3339, body.StartAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid start_at: %v", err), http.StatusBadRequest)
		return
	}
	items, err := h.repo.ListQueueWithTitle(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for _, item := range items {
		if item.Status != "queued" {
			continue
		}
		publishAt := t.UTC().Format(time.RFC3339)
		_ = h.repo.SetSchedule(r.Context(), item.ID, publishAt, "{}")
		t = t.Add(time.Duration(body.IntervalMinutes) * time.Minute)
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostUploadNow handles POST /api/queue/upload-now
// Sets publish_at = now() for all queued items with no publish_at, then triggers the worker.
func (h *QueueHandler) PostUploadNow(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format(time.RFC3339)
	items, err := h.repo.ListQueueWithTitle(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for _, item := range items {
		if item.Status != "queued" {
			continue
		}
		// Only set publish_at if not already scheduled (or if scheduled in future)
		if !item.PublishAt.Valid || item.PublishAt.String > now {
			_ = h.repo.SetSchedule(r.Context(), item.ID, now, "{}")
		}
	}
	if h.triggerUpload != nil {
		go h.triggerUpload()
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostSaveLocal handles POST /api/queue/save-local — no-op (files are already saved during upload confirm)
func (h *QueueHandler) PostSaveLocal(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// GetThumbnail handles GET /api/queue/{id}/thumbnail — serves the thumbnail file
func (h *QueueHandler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	u, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	thumbPath := ""
	if u.LocalThumbnailPath.Valid && u.LocalThumbnailPath.String != "" {
		thumbPath = u.LocalThumbnailPath.String
	}

	// Fallback 1: clip's thumbnail_path via join query
	if thumbPath == "" {
		all, _ := h.repo.ListQueueWithTitle(r.Context())
		for _, q := range all {
			if q.ID == id && q.ThumbnailPath != "" {
				thumbPath = q.ThumbnailPath
				break
			}
		}
	}

	// Fallback 2: thumbnail.jpg alongside local_video_path (SaveToQueue layout)
	if thumbPath == "" && u.LocalVideoPath.Valid && u.LocalVideoPath.String != "" {
		candidate := filepath.Join(filepath.Dir(u.LocalVideoPath.String), "thumbnail.jpg")
		if _, statErr := os.Stat(candidate); statErr == nil {
			thumbPath = candidate
		}
	}

	if thumbPath == "" {
		http.Error(w, "no thumbnail", http.StatusNotFound)
		return
	}
	if _, statErr := os.Stat(thumbPath); statErr != nil {
		http.Error(w, "thumbnail file not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, thumbPath)
}

// queueParseID extracts a named path parameter as int64.
func queueParseID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
