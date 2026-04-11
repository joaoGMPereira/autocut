package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/effects"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// EffectsHandler handles the Video Effects Pipeline endpoints.
// Pattern: POST → 202 {job_id}, GET /{id}/stream → SSE progress.
type EffectsHandler struct {
	svc *effects.EffectsService
	hub *hub.SSEHub
	log *slog.Logger
}

// NewEffectsHandler creates an EffectsHandler.
func NewEffectsHandler(svc *effects.EffectsService, h *hub.SSEHub) *EffectsHandler {
	return &EffectsHandler{
		svc: svc,
		hub: h,
		log: slog.With("component", "api", "handler", "effects"),
	}
}

// ── Request types ─────────────────────────────────────────────────────────────

type antiDupRequest struct {
	VideoPath   string `json:"video_path"`
	CaptionText string `json:"caption_text"`
	OutputPath  string `json:"output_path"`
}

type speedRequest struct {
	VideoPath  string                `json:"video_path"`
	Segments   []effects.SpeedSegment `json:"segments"`
	OutputPath string                `json:"output_path"`
}

type transitionRequest struct {
	ClipPaths          []string               `json:"clip_paths"`
	TransitionType     effects.TransitionType `json:"transition_type"`
	TransitionDuration float64                `json:"transition_duration"`
	OutputPath         string                 `json:"output_path"`
}

type textRequest struct {
	VideoPath  string               `json:"video_path"`
	Text       string               `json:"text"`
	FontName   string               `json:"font_name"`
	FontSize   int                  `json:"font_size"`
	Color      string               `json:"color"`
	Position   effects.TextPosition `json:"position"`
	StartSec   float64              `json:"start_sec"`
	EndSec     float64              `json:"end_sec"`
	OutputPath string               `json:"output_path"`
}

type overlayRequest struct {
	VideoPath  string               `json:"video_path"`
	ImagePath  string               `json:"image_path"`
	Position   effects.TextPosition `json:"position"`
	Opacity    float64              `json:"opacity"`
	Scale      float64              `json:"scale"`
	OutputPath string               `json:"output_path"`
}

// ── AntiDup ──────────────────────────────────────────────────────────────────

// PostAntiDup handles POST /api/effects/anti-dup.
// Returns 202 {"job_id": "<id>"} immediately; progress via SSE.
func (h *EffectsHandler) PostAntiDup(w http.ResponseWriter, r *http.Request) {
	var req antiDupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.VideoPath == "" || req.OutputPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path and output_path are required")
		return
	}

	jobID := newJobID()
	h.log.Info("anti-dup job started", "jobID", jobID, "input", req.VideoPath)
	go h.runAntiDup(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *EffectsHandler) runAntiDup(jobID string, req antiDupRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "progress",
		Data: map[string]string{"status": "running", "progress": "0%", "message": "Applying anti-dup effect..."},
	})

	if err := h.svc.AntiDup(req.VideoPath, req.CaptionText, req.OutputPath); err != nil {
		h.log.Error("anti-dup failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "error",
			Data: map[string]string{"message": err.Error()},
		})
		return
	}

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{"output": req.OutputPath, "success": true},
	})
}

// ── SpeedZones ────────────────────────────────────────────────────────────────

// PostSpeed handles POST /api/effects/speed.
// Returns 202 {"job_id": "<id>"} immediately; progress via SSE.
func (h *EffectsHandler) PostSpeed(w http.ResponseWriter, r *http.Request) {
	var req speedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.VideoPath == "" || req.OutputPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path and output_path are required")
		return
	}
	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "missing_field", "segments must not be empty")
		return
	}

	jobID := newJobID()
	h.log.Info("speed-zones job started", "jobID", jobID, "input", req.VideoPath, "segments", len(req.Segments))
	go h.runSpeed(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *EffectsHandler) runSpeed(jobID string, req speedRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "progress",
		Data: map[string]string{"status": "running", "progress": "0%", "message": "Applying speed zones..."},
	})

	if err := h.svc.SpeedZones(req.VideoPath, req.Segments, req.OutputPath); err != nil {
		h.log.Error("speed-zones failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "error",
			Data: map[string]string{"message": err.Error()},
		})
		return
	}

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{"output": req.OutputPath, "success": true},
	})
}

// ── Transition ────────────────────────────────────────────────────────────────

// PostTransition handles POST /api/effects/transition.
// Returns 202 {"job_id": "<id>"} immediately; progress via SSE.
func (h *EffectsHandler) PostTransition(w http.ResponseWriter, r *http.Request) {
	var req transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if len(req.ClipPaths) < 2 {
		writeError(w, http.StatusBadRequest, "missing_field", "clip_paths must contain at least 2 clips")
		return
	}
	if req.OutputPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "output_path is required")
		return
	}
	if req.TransitionDuration <= 0 {
		req.TransitionDuration = 0.5 // default 500ms transition
	}

	jobID := newJobID()
	h.log.Info("transition job started", "jobID", jobID, "clips", len(req.ClipPaths), "type", req.TransitionType)
	go h.runTransition(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *EffectsHandler) runTransition(jobID string, req transitionRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "progress",
		Data: map[string]string{"status": "running", "progress": "0%", "message": "Applying transition..."},
	})

	if err := h.svc.Transition(req.ClipPaths, req.TransitionType, req.TransitionDuration, req.OutputPath); err != nil {
		h.log.Error("transition failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "error",
			Data: map[string]string{"message": err.Error()},
		})
		return
	}

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{"output": req.OutputPath, "success": true},
	})
}

// ── TextOverlay ───────────────────────────────────────────────────────────────

// PostText handles POST /api/effects/text.
// Returns 202 {"job_id": "<id>"} immediately; progress via SSE.
func (h *EffectsHandler) PostText(w http.ResponseWriter, r *http.Request) {
	var req textRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.VideoPath == "" || req.OutputPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path and output_path are required")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "text is required")
		return
	}
	if req.FontSize <= 0 {
		req.FontSize = 24
	}
	if req.Color == "" {
		req.Color = "white"
	}

	jobID := newJobID()
	h.log.Info("text-overlay job started", "jobID", jobID, "input", req.VideoPath)
	go h.runText(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *EffectsHandler) runText(jobID string, req textRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "progress",
		Data: map[string]string{"status": "running", "progress": "0%", "message": "Applying text overlay..."},
	})

	if err := h.svc.TextOverlay(
		req.VideoPath, req.Text, req.FontName,
		req.FontSize, req.Color, req.Position,
		req.StartSec, req.EndSec, req.OutputPath,
	); err != nil {
		h.log.Error("text-overlay failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "error",
			Data: map[string]string{"message": err.Error()},
		})
		return
	}

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{"output": req.OutputPath, "success": true},
	})
}

// ── Overlay ───────────────────────────────────────────────────────────────────

// PostOverlay handles POST /api/effects/overlay.
// Returns 202 {"job_id": "<id>"} immediately; progress via SSE.
func (h *EffectsHandler) PostOverlay(w http.ResponseWriter, r *http.Request) {
	var req overlayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.VideoPath == "" || req.ImagePath == "" || req.OutputPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path, image_path, and output_path are required")
		return
	}
	if req.Opacity <= 0 || req.Opacity > 1 {
		req.Opacity = 1.0
	}
	if req.Scale <= 0 {
		req.Scale = 1.0
	}

	jobID := newJobID()
	h.log.Info("overlay job started", "jobID", jobID, "input", req.VideoPath, "image", req.ImagePath)
	go h.runOverlay(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *EffectsHandler) runOverlay(jobID string, req overlayRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "progress",
		Data: map[string]string{"status": "running", "progress": "0%", "message": "Applying overlay..."},
	})

	if err := h.svc.Overlay(
		req.VideoPath, req.ImagePath, req.Position,
		req.Opacity, req.Scale, req.OutputPath,
	); err != nil {
		h.log.Error("overlay failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "error",
			Data: map[string]string{"message": err.Error()},
		})
		return
	}

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{"output": req.OutputPath, "success": true},
	})
}

// ── SSE Stream ───────────────────────────────────────────────────────────────

// GetEffectStream handles GET /api/effects/{id}/stream.
// Subscribes the caller to SSE events for the given job ID.
func (h *EffectsHandler) GetEffectStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}
