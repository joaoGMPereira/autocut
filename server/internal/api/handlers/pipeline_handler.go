package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/pipeline"
)

// PipelineHandler handles all pipeline run CRUD, gate advances, and SSE streaming.
type PipelineHandler struct {
	repo *pipeline.Repo
	exec *pipeline.Executor
	hub  *hub.SSEHub
	db   *sql.DB
	log  *slog.Logger
}

// NewPipelineHandler creates a PipelineHandler.
func NewPipelineHandler(
	repo *pipeline.Repo,
	exec *pipeline.Executor,
	h *hub.SSEHub,
	db *sql.DB,
) *PipelineHandler {
	return &PipelineHandler{
		repo: repo,
		exec: exec,
		hub:  h,
		db:   db,
		log:  slog.With("component", "api", "handler", "pipeline"),
	}
}

// ── POST /api/pipeline/runs ───────────────────────────────────────────────────

func (h *PipelineHandler) PostRun(w http.ResponseWriter, r *http.Request) {
	runID, err := h.repo.CreateRun(r.Context(), nil, "", pipeline.StateWaitingURL)
	if err != nil {
		h.log.Error("create run failed", "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", "failed to create run")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"run_id": runID,
		"state":  pipeline.StateWaitingURL,
		"job_id": pipeline.PipelineJobID(runID),
	})
}

// ── GET /api/pipeline/runs ────────────────────────────────────────────────────

func (h *PipelineHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 20
	offset := 0
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	var channelID *int64
	if v := q.Get("channel_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			channelID = &n
		}
	}
	runs, total, err := h.repo.ListRuns(r.Context(), limit, offset, channelID)
	if err != nil {
		h.log.Error("list runs failed", "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", "failed to list runs")
		return
	}
	if runs == nil {
		runs = []*pipeline.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"runs":  runs,
		"total": total,
	})
}

// ── GET /api/pipeline/runs/{id} ───────────────────────────────────────────────

func (h *PipelineHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		h.log.Error("get run failed", "runID", id, "err", err)
		writeError(w, http.StatusNotFound, "not_found", "run not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"run": run})
}

// ── DELETE /api/pipeline/runs/{id} ───────────────────────────────────────────

func (h *PipelineHandler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	if err := h.repo.DeleteRun(r.Context(), id); err != nil {
		h.log.Error("delete run failed", "runID", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", "failed to delete run")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── GET /api/pipeline/runs/{id}/stream ───────────────────────────────────────

func (h *PipelineHandler) GetRunStream(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	h.hub.ServeSSE(w, r, pipeline.PipelineJobID(id))
}

// ── POST /api/pipeline/runs/{id}/advance ─────────────────────────────────────

// PostAdvance handles:
// WAITING_URL  → EventURLSubmitted  (body: {url, channel_id})
// WAITING_MODE → EventModeSubmitted (body: {mode_config: {...}})
func (h *PipelineHandler) PostAdvance(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}

	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		h.log.Error("get run failed", "runID", id, "err", err)
		writeError(w, http.StatusNotFound, "not_found", "run not found")
		return
	}

	var req pipeline.AdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	switch run.State {
	case pipeline.StateWaitingURL:
		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "missing_field", "url is required")
			return
		}
		if err := h.repo.UpdateRunURL(r.Context(), id, req.URL, req.ChannelID); err != nil {
			h.log.Error("update run url failed", "runID", id, "err", err)
			writeError(w, http.StatusInternalServerError, "db_error", "failed to update url")
			return
		}
		next, _ := pipeline.Advance(run.State, run.Mode, pipeline.EventURLSubmitted)
		if err := h.repo.UpdateRunState(r.Context(), id, next, ""); err != nil {
			h.log.Error("update run state failed", "runID", id, "err", err)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": next})

	case pipeline.StateWaitingMode:
		if req.Mode == nil {
			writeError(w, http.StatusBadRequest, "missing_field", "mode_config is required")
			return
		}
		modeConfigJSON, err := json.Marshal(req.Mode)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid mode_config")
			return
		}
		if err := h.repo.UpdateRunMode(r.Context(), id, req.Mode.Mode, string(modeConfigJSON)); err != nil {
			h.log.Error("update run mode failed", "runID", id, "err", err)
			writeError(w, http.StatusInternalServerError, "db_error", "failed to update mode")
			return
		}
		next, _ := pipeline.Advance(run.State, req.Mode.Mode, pipeline.EventModeSubmitted)
		if err := h.repo.UpdateRunState(r.Context(), id, next, ""); err != nil {
			h.log.Error("update run state failed", "runID", id, "err", err)
		}
		// Fire goroutine
		go h.exec.RunExecuting(context.Background(), id)
		writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": next})

	default:
		writeError(w, http.StatusConflict, "invalid_state",
			"advance not valid in state "+string(run.State))
	}
}

// ── POST /api/pipeline/runs/{id}/gates/review-highlights ─────────────────────

func (h *PipelineHandler) PostGateReviewHighlights(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "run not found")
		return
	}
	if run.State != pipeline.StateWaitingReviewHighlights {
		writeError(w, http.StatusConflict, "invalid_state", "run is not at WAITING_REVIEW_HIGHLIGHTS")
		return
	}

	var req pipeline.ReviewHighlightsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	for _, u := range req.Highlights {
		if err := h.repo.UpdateHighlight(r.Context(), u.ID, u.AdjStartSec, u.AdjEndSec, u.IsSelected); err != nil {
			h.log.Error("update highlight failed", "id", u.ID, "err", err)
		}
	}

	next, _ := pipeline.Advance(run.State, run.Mode, pipeline.EventHighlightsReviewed)
	if err := h.repo.UpdateRunState(r.Context(), id, next, ""); err != nil {
		h.log.Error("update run state failed", "runID", id, "err", err)
	}
	go h.exec.RunGeneratingClips(context.Background(), id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": next})
}

// ── POST /api/pipeline/runs/{id}/gates/thumbnail-config ──────────────────────

func (h *PipelineHandler) PostGateThumbnailConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "run not found")
		return
	}
	if run.State != pipeline.StateWaitingThumbnailConfig {
		writeError(w, http.StatusConflict, "invalid_state", "run is not at WAITING_THUMBNAIL_CONFIG")
		return
	}

	var req pipeline.ThumbnailConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	// Apply per-clip overrides
	for _, ov := range req.ClipOverrides {
		if err := h.repo.UpdateClipThumbnailStyle(r.Context(), ov.ClipID, ov.Style); err != nil {
			h.log.Error("update clip thumbnail style failed", "clipID", ov.ClipID, "err", err)
		}
	}

	next, _ := pipeline.Advance(run.State, run.Mode, pipeline.EventThumbnailConfigSubmitted)
	if err := h.repo.UpdateRunState(r.Context(), id, next, ""); err != nil {
		h.log.Error("update run state failed", "runID", id, "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": next})
}

// ── POST /api/pipeline/runs/{id}/gates/review-metadata ───────────────────────

func (h *PipelineHandler) PostGateReviewMetadata(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "run not found")
		return
	}
	if run.State != pipeline.StateWaitingReviewMetadata {
		writeError(w, http.StatusConflict, "invalid_state", "run is not at WAITING_REVIEW_METADATA")
		return
	}

	var req pipeline.ReviewMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	for _, u := range req.Clips {
		if err := h.repo.UpdateClipMetadata(r.Context(), u.ID, u.Title, u.Description, u.Tags); err != nil {
			h.log.Error("update clip metadata failed", "id", u.ID, "err", err)
		}
	}

	next, _ := pipeline.Advance(run.State, run.Mode, pipeline.EventMetadataReviewed)
	if err := h.repo.UpdateRunState(r.Context(), id, next, ""); err != nil {
		h.log.Error("update run state failed", "runID", id, "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": next})
}

// ── POST /api/pipeline/runs/{id}/gates/review-clips ──────────────────────────

func (h *PipelineHandler) PostGateReviewClips(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "run not found")
		return
	}
	if run.State != pipeline.StateWaitingReviewClips {
		writeError(w, http.StatusConflict, "invalid_state", "run is not at WAITING_REVIEW_CLIPS")
		return
	}

	var req pipeline.ReviewClipsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	selectedSet := make(map[int64]bool, len(req.SelectedIDs))
	for _, sid := range req.SelectedIDs {
		selectedSet[sid] = true
	}
	clips, _ := h.repo.GetClips(r.Context(), id)
	for _, c := range clips {
		if err := h.repo.UpdateClipSelected(r.Context(), c.ID, selectedSet[c.ID]); err != nil {
			h.log.Error("update clip selected failed", "clipID", c.ID, "err", err)
		}
	}

	next, _ := pipeline.Advance(run.State, run.Mode, pipeline.EventClipsReviewed)
	if err := h.repo.UpdateRunState(r.Context(), id, next, ""); err != nil {
		h.log.Error("update run state failed", "runID", id, "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": next})
}

// ── POST /api/pipeline/runs/{id}/gates/upload-confirm ────────────────────────

func (h *PipelineHandler) PostGateUploadConfirm(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "run not found")
		return
	}
	if run.State != pipeline.StateWaitingUploadConfirm {
		writeError(w, http.StatusConflict, "invalid_state", "run is not at WAITING_UPLOAD_CONFIRM")
		return
	}

	var req pipeline.UploadConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	privacy := req.Privacy
	if privacy == "" {
		privacy = "private"
	}

	next, _ := pipeline.Advance(run.State, run.Mode, pipeline.EventUploadConfirmed)
	if err := h.repo.UpdateRunState(r.Context(), id, next, ""); err != nil {
		h.log.Error("update run state failed", "runID", id, "err", err)
	}
	go h.exec.RunUploading(context.Background(), id, privacy)
	writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": next})
}

// ── POST /api/pipeline/runs/{id}/cancel ──────────────────────────────────────

func (h *PipelineHandler) PostCancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	if err := h.repo.FinishRun(r.Context(), id, pipeline.StateCancelled, "cancelled by user"); err != nil {
		h.log.Error("cancel run failed", "runID", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", "failed to cancel run")
		return
	}
	publishCancelled(h.hub, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"run_id": id, "state": pipeline.StateCancelled})
}

// ── GET /api/pipeline/runs/{id}/highlights ────────────────────────────────────

func (h *PipelineHandler) GetHighlights(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	highlights, err := h.repo.GetHighlights(r.Context(), id)
	if err != nil {
		h.log.Error("get highlights failed", "runID", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", "failed to get highlights")
		return
	}
	if highlights == nil {
		highlights = []pipeline.Highlight{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"highlights": highlights})
}

// ── GET /api/pipeline/runs/{id}/clips ─────────────────────────────────────────

func (h *PipelineHandler) GetClips(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	clips, err := h.repo.GetClips(r.Context(), id)
	if err != nil {
		h.log.Error("get clips failed", "runID", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", "failed to get clips")
		return
	}
	if clips == nil {
		clips = []pipeline.Clip{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"clips": clips})
}

// ── helper ────────────────────────────────────────────────────────────────────

func parseRunID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	v := r.PathValue("id")
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be an integer")
		return 0, false
	}
	return id, true
}

// publishCancelled is a local helper to publish SSE from the handler.
func publishCancelled(h *hub.SSEHub, runID int64) {
	h.Publish(pipeline.PipelineJobID(runID), hub.SSEEvent{
		Type: pipeline.SSEEventCancelled,
		Data: map[string]interface{}{
			"run_id": runID,
			"state":  pipeline.StateCancelled,
		},
	})
}
