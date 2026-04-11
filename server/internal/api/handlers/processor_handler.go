package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/processor"
	"github.com/joaoGMPereira/autocut/server/internal/thumbnail"
)

// ProcessorHandler handles FFmpeg cut, shorts, and optimizer requests.
type ProcessorHandler struct {
	hub          *hub.SSEHub
	proc         *processor.FFmpegProcessor
	shorts       *processor.ShortsGenerator
	optimizer    *processor.VideoOptimizerProcessor
	thumbGen     *thumbnail.ThumbnailGenerator
	pipelineRepo *database.PipelineRunRepo
	log          *slog.Logger
}

// NewProcessorHandler creates a ProcessorHandler.
// thumbGen may be nil; when nil, per-short thumbnail generation is skipped.
func NewProcessorHandler(
	h *hub.SSEHub,
	proc *processor.FFmpegProcessor,
	shorts *processor.ShortsGenerator,
	optimizer *processor.VideoOptimizerProcessor,
	pipelineRepo *database.PipelineRunRepo,
) *ProcessorHandler {
	return &ProcessorHandler{
		hub:          h,
		proc:         proc,
		shorts:       shorts,
		optimizer:    optimizer,
		pipelineRepo: pipelineRepo,
		log:          slog.With("component", "api", "handler", "processor"),
	}
}

// WithThumbnailGenerator attaches a ThumbnailGenerator for per-short thumbnails.
// Call this after NewProcessorHandler when a generator is available.
func (h *ProcessorHandler) WithThumbnailGenerator(g *thumbnail.ThumbnailGenerator) *ProcessorHandler {
	h.thumbGen = g
	return h
}

// --- Cut ---

type cutRequest struct {
	Input     string  `json:"input"`
	StartSec  float64 `json:"start_sec"`
	EndSec    float64 `json:"end_sec"`
	Output    string  `json:"output"`
	SessionID string  `json:"session_id"`
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
	data := map[string]interface{}{"output": req.Output, "success": true}
	h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
	if req.SessionID != "" {
		persistStepOutput(h.pipelineRepo, req.SessionID, "cut", data)
	}
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

// shortsRequest is the request body for POST /api/shorts.
// All new fields (FP-006) are optional — omitting them preserves existing behaviour.
type shortsRequest struct {
	// Existing fields (backward compatible)
	Input     string `json:"input"`
	Output    string `json:"output"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SessionID string `json:"session_id"`

	// FP-006: Blur background composite (LL-005)
	AddBlurBackground bool `json:"add_blur_background"`

	// FP-006: Word-by-word caption burn-in
	AddCaptions    bool   `json:"add_captions"`
	TranscriptPath string `json:"transcript_path"`

	// FP-006: Multi-language support (PT default)
	Language string `json:"language"`

	// FP-006: Per-short thumbnail generation
	GenerateThumbnails bool `json:"generate_thumbnails"`

	// FP-006: Duration filter (0 = no limit)
	MinDuration float64 `json:"min_duration"`
	MaxDuration float64 `json:"max_duration"`
}

// transcriptSegmentOnDisk matches the JSON format written by the transcript service.
// Used to load transcript from disk in the handler — converts to processor.SubtitleSegment.
type transcriptSegmentOnDisk struct {
	StartSec float64 `json:"start"`
	EndSec   float64 `json:"end"`
	Text     string  `json:"text"`
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
	h.log.Info("shorts job started", "jobID", jobID, "input", req.Input,
		"blur", req.AddBlurBackground, "captions", req.AddCaptions,
		"lang", req.Language, "thumbs", req.GenerateThumbnails)
	go h.runShorts(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *ProcessorHandler) runShorts(jobID string, req shortsRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "generating"}})

	lang := req.Language
	if lang == "" {
		lang = "pt"
	}

	cfg := processor.ShortsConfig{
		Width:             req.Width,
		Height:            req.Height,
		AddBlurBackground: req.AddBlurBackground,
		Language:          lang,
	}

	var genErr error

	if req.AddCaptions && req.TranscriptPath != "" {
		// Load transcript segments from disk and burn in word-by-word captions.
		segments, err := h.loadTranscriptSegments(req.TranscriptPath)
		if err != nil {
			h.log.Error("shorts: load transcript failed", "jobID", jobID, "path", req.TranscriptPath, "err", err)
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": "load transcript: " + err.Error()}})
			return
		}
		genErr = h.shorts.GenerateCaptionedShort(req.Input, segments, req.Output, cfg)
	} else {
		genErr = h.shorts.Generate(req.Input, cfg, req.Output)
	}

	if genErr != nil {
		h.log.Error("shorts generation failed", "jobID", jobID, "err", genErr)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": genErr.Error()}})
		return
	}

	// FP-006: Engagement scoring — uses zero highlight confidence when not
	// called from a highlight pipeline; caller can inject via future extension.
	engScore := ai.ScoreEngagement(0.5, nil, videoDurationEstimate(req.MinDuration, req.MaxDuration), lang)
	h.log.Debug("shorts engagement scored", "jobID", jobID, "score", engScore.Score,
		"factors", engScore.Factors)

	// FP-006: Per-short thumbnail generation.
	thumbPath := ""
	if req.GenerateThumbnails && h.thumbGen != nil {
		ext := filepath.Ext(req.Output)
		thumbPath = strings.TrimSuffix(req.Output, ext) + "_thumb.jpg"
		thumbCfg := thumbnail.ShortsThumbnailConfig{
			VideoPath: req.Output,
			Width:     1080,
			Height:    1920,
		}
		if cfg.Width > 0 {
			thumbCfg.Width = cfg.Width
		}
		if cfg.Height > 0 {
			thumbCfg.Height = cfg.Height
		}
		if err := h.thumbGen.GenerateShortsThumbnail(thumbCfg, thumbPath); err != nil {
			h.log.Warn("shorts: thumbnail generation failed (non-fatal)",
				"jobID", jobID, "err", err)
			thumbPath = "" // do not surface a broken path
		} else {
			h.log.Info("shorts: thumbnail generated", "jobID", jobID, "path", thumbPath)
		}
	}

	data := map[string]interface{}{
		"output":             req.Output,
		"success":            true,
		"engagement_score":   engScore.Score,
		"engagement_factors": engScore.Factors,
		"thumbnail_path":     thumbPath,
	}
	h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
	if req.SessionID != "" {
		persistStepOutput(h.pipelineRepo, req.SessionID, "shorts", data)
	}
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
	SessionID        string  `json:"session_id"`
}

// PostOptimize handles POST /api/optimize.
func (h *ProcessorHandler) PostOptimize(w http.ResponseWriter, r *http.Request) {
	var req optimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Input == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "input is required")
		return
	}
	if req.Output == "" {
		ext := filepath.Ext(req.Input)
		req.Output = strings.TrimSuffix(req.Input, ext) + "_optimized" + ext
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
	data := map[string]interface{}{
		"output":           req.Output,
		"removed_silences": result.RemovedSilences,
		"original_sec":     result.OriginalDuration.Seconds(),
		"final_sec":        result.FinalDuration.Seconds(),
		"success":          true,
	}
	h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
	if req.SessionID != "" {
		persistStepOutput(h.pipelineRepo, req.SessionID, "optimize", data)
	}
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

// --- Private helpers ---

// loadTranscriptSegments reads a transcript JSON file from disk and converts
// it to []processor.SubtitleSegment for caption burn-in.
// The transcript file is expected to be an array of objects with
// start (float64 seconds), end (float64 seconds), and text (string) fields.
func (h *ProcessorHandler) loadTranscriptSegments(path string) ([]processor.SubtitleSegment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		h.log.Error("loadTranscriptSegments: read failed", "path", path, "err", err)
		return nil, err
	}

	var raw []transcriptSegmentOnDisk
	if err := json.Unmarshal(data, &raw); err != nil {
		h.log.Error("loadTranscriptSegments: unmarshal failed", "path", path, "err", err)
		return nil, err
	}

	segs := make([]processor.SubtitleSegment, 0, len(raw))
	for _, t := range raw {
		segs = append(segs, processor.SubtitleSegment{
			Start: time.Duration(t.StartSec * float64(time.Second)),
			End:   time.Duration(t.EndSec * float64(time.Second)),
			Text:  t.Text,
		})
	}
	return segs, nil
}

// videoDurationEstimate returns a representative duration in seconds for engagement
// scoring when the exact video duration is not available in the handler.
// Uses the midpoint of [minDuration, maxDuration] when both are set, or 30s default.
func videoDurationEstimate(minDur, maxDur float64) float64 {
	if minDur > 0 && maxDur > 0 {
		return (minDur + maxDur) / 2.0
	}
	if maxDur > 0 {
		return maxDur
	}
	if minDur > 0 {
		return minDur
	}
	return 30.0 // sensible default for a short
}
