package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/processor"
)

// ProcessorHandler handles FFmpeg cut, shorts, and optimizer requests.
type ProcessorHandler struct {
	hub       *hub.SSEHub
	proc      *processor.FFmpegProcessor
	shorts    *processor.ShortsGenerator
	optimizer *processor.VideoOptimizerProcessor
	log       *slog.Logger
}

// NewProcessorHandler creates a ProcessorHandler.
func NewProcessorHandler(
	h *hub.SSEHub,
	proc *processor.FFmpegProcessor,
	shorts *processor.ShortsGenerator,
	optimizer *processor.VideoOptimizerProcessor,
) *ProcessorHandler {
	return &ProcessorHandler{
		hub:       h,
		proc:      proc,
		shorts:    shorts,
		optimizer: optimizer,
		log:       slog.With("component", "api", "handler", "processor"),
	}
}

// --- Cut ---

type cutRequest struct {
	Input    string  `json:"input"`
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
	Output   string  `json:"output"`
}

// PostCut handles POST /api/cut.
func (h *ProcessorHandler) PostCut(w http.ResponseWriter, r *http.Request) {
	var req cutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Input == "" || req.Output == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "input and output are required")
		return
	}

	jobID := newJobID()
	h.log.Info("cut job started", "jobID", jobID, "input", req.Input)
	go h.runCut(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *ProcessorHandler) runCut(jobID string, req cutRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "cutting"}})

	start := time.Duration(req.StartSec * float64(time.Second))
	end := time.Duration(req.EndSec * float64(time.Second))

	if err := h.proc.CutClip(req.Input, start, end, req.Output); err != nil {
		h.log.Error("cut failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
		return
	}
	h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]string{"output": req.Output}})
}

// GetCutStream handles GET /api/cut/{id}/stream.
func (h *ProcessorHandler) GetCutStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}

// --- Shorts ---

type shortsRequest struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// PostShorts handles POST /api/shorts.
func (h *ProcessorHandler) PostShorts(w http.ResponseWriter, r *http.Request) {
	var req shortsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Input == "" || req.Output == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "input and output are required")
		return
	}

	jobID := newJobID()
	h.log.Info("shorts job started", "jobID", jobID, "input", req.Input)
	go h.runShorts(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *ProcessorHandler) runShorts(jobID string, req shortsRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "generating"}})

	cfg := processor.ShortsConfig{
		Width:  req.Width,
		Height: req.Height,
	}

	if err := h.shorts.Generate(req.Input, cfg, req.Output); err != nil {
		h.log.Error("shorts generation failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
		return
	}
	h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]string{"output": req.Output}})
}

// GetShortsStream handles GET /api/shorts/{id}/stream.
func (h *ProcessorHandler) GetShortsStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}

// --- Optimize ---

type optimizeRequest struct {
	Input            string  `json:"input"`
	Output           string  `json:"output"`
	SilenceThreshold float64 `json:"silence_threshold"`
}

// PostOptimize handles POST /api/optimize.
func (h *ProcessorHandler) PostOptimize(w http.ResponseWriter, r *http.Request) {
	var req optimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Input == "" || req.Output == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "input and output are required")
		return
	}

	jobID := newJobID()
	h.log.Info("optimize job started", "jobID", jobID, "input", req.Input)
	go h.runOptimize(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *ProcessorHandler) runOptimize(jobID string, req optimizeRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "optimizing"}})

	cfg := processor.OptimizerConfig{
		SilenceThreshold: req.SilenceThreshold,
	}

	result, err := h.optimizer.Optimize(req.Input, cfg, req.Output)
	if err != nil {
		h.log.Error("optimize failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
		return
	}
	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{
			"output":           req.Output,
			"removed_silences": result.RemovedSilences,
			"original_sec":     result.OriginalDuration.Seconds(),
			"final_sec":        result.FinalDuration.Seconds(),
		},
	})
}

// GetOptimizeStream handles GET /api/optimize/{id}/stream.
func (h *ProcessorHandler) GetOptimizeStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}
