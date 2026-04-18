package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/pipeline"
)

// PipelineHandler handles HTTP requests for pipeline run management.
// It parses requests and delegates all business logic to pipeline.Service.
type PipelineHandler struct {
	svc *pipeline.Service
	hub *hub.SSEHub
	log *slog.Logger
}

// NewPipelineHandler constructs a PipelineHandler.
func NewPipelineHandler(svc *pipeline.Service, h *hub.SSEHub) *PipelineHandler {
	return &PipelineHandler{
		svc: svc,
		hub: h,
		log: slog.With("handler", "pipeline"),
	}
}

// runResponse is the JSON shape returned for a single pipeline run.
// Maps Go PipelineRun fields to the TypeScript Run interface field names.
type runResponse struct {
	ID          int64   `json:"id"`
	ChannelID   *int64  `json:"channel_id"`
	URL         string  `json:"url"`
	Mode        string  `json:"mode"`
	State       string  `json:"state"`
	ActivePhase string  `json:"active_phase"`
	Error       string  `json:"error"`
	VideoPath   string  `json:"video_path"`
	VideoTitle  string  `json:"video_title"`
	DurationSec int64   `json:"duration_sec"`
	StartedAt   int64   `json:"started_at"`
	FinishedAt  *int64  `json:"finished_at"`
}

// PostRuns creates a new pipeline run in WAITING_URL state.
// POST /api/pipeline/runs
func (h *PipelineHandler) PostRuns(w http.ResponseWriter, r *http.Request) {
	run, err := h.svc.Create(r.Context())
	if err != nil {
		h.log.Error("create run failed", "err", err)
		jsonError(w, "failed to create run", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"run_id": run.ID,
		"state":  run.State,
	})
}

// GetRun returns the current state of a pipeline run.
// GET /api/pipeline/runs/{id}
func (h *PipelineHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	run, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, "run not found", http.StatusNotFound)
			return
		}
		h.log.Error("get run failed", "run_id", id, "err", err)
		jsonError(w, "failed to get run", http.StatusInternalServerError)
		return
	}
	resp := runResponse{
		ID:          run.ID,
		URL:         run.URL,
		Mode:        run.Mode,
		State:       run.State,
		ActivePhase: run.ActivePhase,
		Error:       run.Error,
		VideoPath:   run.VideoPath,
		VideoTitle:  run.VideoTitle,
		DurationSec: run.DurationSec,
		StartedAt:   run.StartedAt,
	}
	if run.ChannelID.Valid {
		v := run.ChannelID.Int64
		resp.ChannelID = &v
	}
	if run.FinishedAt.Valid {
		v := run.FinishedAt.Int64
		resp.FinishedAt = &v
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"run": resp})
}

// GetRunStream opens an SSE stream for a pipeline run.
// GET /api/pipeline/runs/{id}/stream
func (h *PipelineHandler) GetRunStream(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	jobKey := fmt.Sprintf("%d", id)
	h.hub.ServeSSE(w, r, jobKey)
}

// PostAdvance resolves the current gate state. Polymorphic — behavior depends on run's state.
// POST /api/pipeline/runs/{id}/advance
func (h *PipelineHandler) PostAdvance(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	var req pipeline.AdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	result, err := h.svc.Advance(r.Context(), id, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errBadRequest(err)) {
			status = http.StatusBadRequest
		}
		h.log.Error("advance run failed", "run_id", id, "err", err)
		jsonError(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"state":        result.State,
		"video_reused": result.VideoReused,
	})
}

// PostCancel transitions a run to CANCELLED.
// POST /api/pipeline/runs/{id}/cancel
func (h *PipelineHandler) PostCancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	newState, err := h.svc.Cancel(r.Context(), id)
	if err != nil {
		h.log.Error("cancel run failed", "run_id", id, "err", err)
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"state": newState})
}

// PostRedownload forces a fresh video download for a run in WAITING_MODE state.
// POST /api/pipeline/runs/{id}/redownload
func (h *PipelineHandler) PostRedownload(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Redownload(r.Context(), id); err != nil {
		h.log.Error("redownload failed", "run_id", id, "err", err)
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// GetPriorClips checks if a prior completed run exists for a given URL.
// GET /api/pipeline/runs/prior-clips?url=<encoded>
func (h *PipelineHandler) GetPriorClips(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		jsonError(w, "url query parameter is required", http.StatusBadRequest)
		return
	}
	exists, runID, err := h.svc.HasPriorRunWithClips(r.Context(), rawURL)
	if err != nil {
		h.log.Error("has prior run with clips failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if exists {
		_ = json.NewEncoder(w).Encode(map[string]any{"exists": true, "run_id": runID})
	} else {
		_ = json.NewEncoder(w).Encode(map[string]any{"exists": false})
	}
}

// GetRunClips returns all clips for a pipeline run.
// GET /api/pipeline/runs/{id}/clips
func (h *PipelineHandler) GetRunClips(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	clips, err := h.svc.ListClipsByRun(r.Context(), id)
	if err != nil {
		h.log.Error("list clips failed", "run_id", id, "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"clips": clips})
}

// PostReviewClips saves clip selection + optional text edits, then advances to WAITING_UPLOAD_CONFIRM.
// POST /api/pipeline/runs/{id}/gates/review-clips
func (h *PipelineHandler) PostReviewClips(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRunID(w, r)
	if !ok {
		return
	}
	var req pipeline.ReviewClipsPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	newState, err := h.svc.ReviewClips(r.Context(), id, req)
	if err != nil {
		h.log.Error("review clips failed", "run_id", id, "err", err)
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not in") {
			status = http.StatusConflict
		}
		jsonError(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"state": newState})
}

// parseRunID extracts the {id} path value and writes an error response on failure.
func parseRunID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		jsonError(w, "invalid run id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// errBadRequest is a sentinel — advance returns plain errors; we check content.
func errBadRequest(err error) error { return err }
