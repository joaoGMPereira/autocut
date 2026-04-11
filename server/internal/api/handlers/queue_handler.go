package handlers

import (
	"database/sql"
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

// QueueHandler handles upload queue persistence and scheduling (FP-012 + FP-013).
type QueueHandler struct {
	repo *database.UploadRepo
	log  *slog.Logger
}

// NewQueueHandler creates a QueueHandler.
func NewQueueHandler(db *sql.DB) *QueueHandler {
	logger := slog.Default().With("component", "api", "handler", "queue")
	return &QueueHandler{
		repo: database.NewUploadRepo(db, slog.Default()),
		log:  logger,
	}
}

// QueueItem is the JSON representation of a queue entry returned by the API.
type QueueItem struct {
	ID               int64  `json:"id"`
	ChannelID        int64  `json:"channel_id"`
	VideoPath        string `json:"video_path"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Tags             string `json:"tags"`
	Privacy          string `json:"privacy"`
	Status           string `json:"status"`
	QueueOrder       int    `json:"queue_order"`
	PublishAt        string `json:"publish_at,omitempty"`
	CreatedAt        int64  `json:"created_at"`
}

// metadataFields maps the metadata JSON fields stored in metadata_json.
type metadataFields struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
}

// uploadConfigFields maps the config JSON fields stored in upload_config_json.
type uploadConfigFields struct {
	Privacy string `json:"privacy"`
}

// queueRowToItem converts a database.QueueRow to a QueueItem for the API response.
func queueRowToItem(q database.QueueRow) QueueItem {
	item := QueueItem{
		ID:         q.ID,
		ChannelID:  q.ChannelID,
		VideoPath:  q.LocalVideoPath.String,
		Status:     q.Status,
		QueueOrder: q.QueueOrder,
		CreatedAt:  q.CreatedAt,
	}
	if q.PublishAt.Valid {
		item.PublishAt = q.PublishAt.String
	}

	var meta metadataFields
	if err := json.Unmarshal([]byte(q.MetadataJSON), &meta); err == nil {
		item.Title = meta.Title
		item.Description = meta.Description
		item.Tags = meta.Tags
	}

	var cfg uploadConfigFields
	if err := json.Unmarshal([]byte(q.UploadConfigJSON), &cfg); err == nil {
		item.Privacy = cfg.Privacy
	}
	if item.Privacy == "" {
		item.Privacy = "private"
	}

	return item
}

// GetQueue handles GET /api/queue
// Returns uploads with status in (queued, running, failed) ordered by queue_order.
func (h *QueueHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	rows, err := h.repo.ListQueue(r.Context())
	if err != nil {
		h.log.Error("list queue failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	items := make([]QueueItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, queueRowToItem(row))
	}
	writeJSON(w, http.StatusOK, items)
}

// postQueueRequest is the body for POST /api/queue.
type postQueueRequest struct {
	VideoPath   string `json:"video_path"`
	ChannelID   int64  `json:"channel_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	Privacy     string `json:"privacy"`
}

// PostQueue handles POST /api/queue
// Creates a new upload entry with status=queued.
func (h *QueueHandler) PostQueue(w http.ResponseWriter, r *http.Request) {
	var req postQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.VideoPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path is required")
		return
	}
	if req.ChannelID <= 0 {
		writeError(w, http.StatusBadRequest, "missing_field", "channel_id must be > 0")
		return
	}
	if req.Privacy == "" {
		req.Privacy = "private"
	}

	metaBytes, err := json.Marshal(metadataFields{
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
	})
	if err != nil {
		h.log.Error("marshal metadata failed", "err", err)
		writeError(w, http.StatusInternalServerError, "marshal_failed", err.Error())
		return
	}

	cfgBytes, err := json.Marshal(uploadConfigFields{Privacy: req.Privacy})
	if err != nil {
		h.log.Error("marshal config failed", "err", err)
		writeError(w, http.StatusInternalServerError, "marshal_failed", err.Error())
		return
	}

	id, err := h.repo.CreateQueued(r.Context(), req.ChannelID, req.VideoPath,
		string(metaBytes), string(cfgBytes), 0)
	if err != nil {
		h.log.Error("create queued upload failed", "err", err)
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	h.log.Info("upload queued", "id", id, "channelID", req.ChannelID, "videoPath", req.VideoPath)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":     id,
		"status": "queued",
	})
}

// DeleteQueue handles DELETE /api/queue/{id}
// Removes a queued upload. Returns 409 if status != queued.
func (h *QueueHandler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	upload, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("get upload for delete failed", "id", id, "err", err)
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("upload %d not found", id))
		return
	}
	if upload.Status != "queued" {
		writeError(w, http.StatusConflict, "invalid_status",
			fmt.Sprintf("cannot delete upload with status=%q (only queued allowed)", upload.Status))
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.log.Error("delete upload failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	h.log.Info("upload deleted from queue", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// PostQueueRetry handles POST /api/queue/{id}/retry
// Moves a failed upload back to queued. Returns 409 if status != failed.
func (h *QueueHandler) PostQueueRetry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	upload, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("get upload for retry failed", "id", id, "err", err)
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("upload %d not found", id))
		return
	}
	if upload.Status != "failed" {
		writeError(w, http.StatusConflict, "invalid_status",
			fmt.Sprintf("cannot retry upload with status=%q (only failed allowed)", upload.Status))
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, "queued"); err != nil {
		h.log.Error("retry upload failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "retry_failed", err.Error())
		return
	}

	h.log.Info("upload retried", "id", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":     id,
		"status": "queued",
	})
}

// PostQueueSaveLocal handles POST /api/queue/save-local
// Exports the current queue to data/queue/queue_{timestamp}.json.
func (h *QueueHandler) PostQueueSaveLocal(w http.ResponseWriter, r *http.Request) {
	rows, err := h.repo.ListQueue(r.Context())
	if err != nil {
		h.log.Error("list queue for save-local failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	items := make([]QueueItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, queueRowToItem(row))
	}

	dir := filepath.Join("data", "queue")
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.log.Error("create queue dir failed", "dir", dir, "err", err)
		writeError(w, http.StatusInternalServerError, "dir_failed", err.Error())
		return
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	filePath := filepath.Join(dir, fmt.Sprintf("queue_%s.json", ts))

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		h.log.Error("marshal queue for save-local failed", "err", err)
		writeError(w, http.StatusInternalServerError, "marshal_failed", err.Error())
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		h.log.Error("write queue file failed", "path", filePath, "err", err)
		writeError(w, http.StatusInternalServerError, "write_failed", err.Error())
		return
	}

	h.log.Info("queue saved locally", "path", filePath, "count", len(items))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"file_path": filePath,
		"count":     len(items),
	})
}

// ── FP-013: Scheduling endpoints ─────────────────────────────────────────────

// scheduleRequest is the body for POST /api/queue/{id}/schedule.
type scheduleRequest struct {
	PublishAt string `json:"publish_at"`
	Privacy   string `json:"privacy"`
}

// PostQueueSchedule handles POST /api/queue/{id}/schedule
// Sets publish_at (ISO8601/RFC3339) and privacy for a queued upload.
func (h *QueueHandler) PostQueueSchedule(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.PublishAt == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "publish_at is required")
		return
	}
	if _, err := time.Parse(time.RFC3339, req.PublishAt); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_field",
			fmt.Sprintf("publish_at must be RFC3339 (e.g. 2026-04-01T15:00:00Z): %v", err))
		return
	}
	if req.Privacy == "" {
		req.Privacy = "private"
	}

	// Fetch current upload_config_json, merge privacy into it.
	upload, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("get upload for schedule failed", "id", id, "err", err)
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("upload %d not found", id))
		return
	}

	// Merge privacy into existing config blob.
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(upload.UploadConfigJSON), &cfg); err != nil {
		cfg = make(map[string]interface{})
	}
	cfg["privacy"] = req.Privacy
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		h.log.Error("marshal config for schedule failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "marshal_failed", err.Error())
		return
	}

	if err := h.repo.SetSchedule(r.Context(), id, req.PublishAt, string(cfgBytes)); err != nil {
		h.log.Error("set schedule failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "schedule_failed", err.Error())
		return
	}

	h.log.Info("upload scheduled", "id", id, "publishAt", req.PublishAt, "privacy", req.Privacy)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         id,
		"publish_at": req.PublishAt,
		"privacy":    req.Privacy,
	})
}

// bulkScheduleRequest is the body for POST /api/queue/bulk-schedule.
type bulkScheduleRequest struct {
	IDs             []int64 `json:"ids"`
	StartAt         string  `json:"start_at"`
	IntervalMinutes int     `json:"interval_minutes"`
}

// PostQueueBulkSchedule handles POST /api/queue/bulk-schedule
// Assigns staggered publish times to a list of uploads starting at start_at,
// incrementing by interval_minutes for each subsequent upload.
func (h *QueueHandler) PostQueueBulkSchedule(w http.ResponseWriter, r *http.Request) {
	var req bulkScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "missing_field", "ids must be non-empty")
		return
	}
	if req.IntervalMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "missing_field", "interval_minutes must be > 0")
		return
	}
	if req.StartAt == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "start_at is required")
		return
	}

	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_field",
			fmt.Sprintf("start_at must be RFC3339 (e.g. 2026-04-01T15:00:00Z): %v", err))
		return
	}

	interval := time.Duration(req.IntervalMinutes) * time.Minute
	scheduledCount := 0

	for i, id := range req.IDs {
		publishAt := startAt.Add(time.Duration(i) * interval)
		publishAtStr := publishAt.UTC().Format(time.RFC3339)

		upload, err := h.repo.GetByID(r.Context(), id)
		if err != nil {
			h.log.Warn("bulk-schedule: upload not found, skipping", "id", id, "err", err)
			continue
		}

		var cfg map[string]interface{}
		if err := json.Unmarshal([]byte(upload.UploadConfigJSON), &cfg); err != nil {
			cfg = make(map[string]interface{})
		}
		cfgBytes, err := json.Marshal(cfg)
		if err != nil {
			h.log.Error("bulk-schedule: marshal config failed", "id", id, "err", err)
			continue
		}

		if err := h.repo.SetSchedule(r.Context(), id, publishAtStr, string(cfgBytes)); err != nil {
			h.log.Error("bulk-schedule: set schedule failed", "id", id, "err", err)
			continue
		}

		h.log.Info("bulk-schedule: upload scheduled", "id", id, "publishAt", publishAtStr)
		scheduledCount++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"scheduled_count": scheduledCount,
	})
}

// GetQueueSchedule handles GET /api/queue/schedule
// Returns queued uploads that have a publish_at set, ordered by publish_at.
func (h *QueueHandler) GetQueueSchedule(w http.ResponseWriter, r *http.Request) {
	rows, err := h.repo.ListScheduled(r.Context())
	if err != nil {
		h.log.Error("list scheduled failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	items := make([]QueueItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, queueRowToItem(row))
	}
	writeJSON(w, http.StatusOK, items)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// parseIDParam extracts the "id" path parameter and parses it as int64.
// Writes a 400 error and returns false on failure.
func parseIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.PathValue("id")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id path parameter is required")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", fmt.Sprintf("id must be an integer: %v", err))
		return 0, false
	}
	return id, true
}
