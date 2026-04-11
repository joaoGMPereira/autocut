package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/thumbnail"
)

// ThumbnailHandler handles synchronous thumbnail generation (no SSE).
type ThumbnailHandler struct {
	gen          *thumbnail.ThumbnailGenerator
	pipelineRepo *database.PipelineRunRepo
	bgRepo       *database.ThumbnailBackgroundRepo
	log          *slog.Logger
}

// NewThumbnailHandler creates a ThumbnailHandler.
// bgRepo is used for the "template" strategy to look up stored channel backgrounds.
func NewThumbnailHandler(gen *thumbnail.ThumbnailGenerator, pipelineRepo *database.PipelineRunRepo, bgRepo *database.ThumbnailBackgroundRepo) *ThumbnailHandler {
	return &ThumbnailHandler{
		gen:          gen,
		pipelineRepo: pipelineRepo,
		bgRepo:       bgRepo,
		log:          slog.With("component", "api", "handler", "thumbnail"),
	}
}

type thumbnailRequest struct {
	VideoPath    string `json:"video_path"`
	Template     string `json:"template"`      // "branded" | "centered" (legacy field)
	Strategy     string `json:"strategy"`      // "branded" | "centered" | "template" | "ai"
	Text         string `json:"text"`
	FontColor    string `json:"font_color"`
	FontSize     int    `json:"font_size"`
	Output       string `json:"output"`
	SessionID    string `json:"session_id"`
	BackgroundID int64  `json:"background_id"` // required when strategy="template"
	CustomPrompt string `json:"custom_prompt"` // optional Ollama prompt override for strategy="ai"
}

// effectiveStrategy returns the strategy to use, preferring the new Strategy field
// but falling back to the legacy Template field for backward compatibility.
func (req thumbnailRequest) effectiveStrategy() string {
	if req.Strategy != "" {
		return req.Strategy
	}
	return req.Template
}

// PostThumbnail handles POST /api/thumbnail (synchronous — no SSE).
func (h *ThumbnailHandler) PostThumbnail(w http.ResponseWriter, r *http.Request) {
	var req thumbnailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	strategy := req.effectiveStrategy()

	// video_path is required for all strategies except "template" (which uses background_id).
	if strategy != "template" && req.VideoPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path is required")
		return
	}
	if req.Output == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "output is required")
		return
	}

	var err error
	switch strategy {
	case "template":
		if req.BackgroundID == 0 {
			writeError(w, http.StatusBadRequest, "missing_field", "background_id is required for strategy=template")
			return
		}
		bg, bgErr := h.bgRepo.GetByID(r.Context(), req.BackgroundID)
		if bgErr != nil {
			h.log.Error("background lookup failed", "backgroundID", req.BackgroundID, "err", bgErr)
			writeError(w, http.StatusNotFound, "not_found", "background not found")
			return
		}
		cfg := thumbnail.TemplateConfig{
			BackgroundPath: bg.FilePath,
			Title:          req.Text,
			FontSize:       req.FontSize,
			FontColor:      req.FontColor,
		}
		err = h.gen.GenerateTemplate(cfg, req.Output)

	case "ai":
		cfg := thumbnail.AIThumbnailConfig{
			VideoPath:    req.VideoPath,
			CustomPrompt: req.CustomPrompt,
			FontColor:    req.FontColor,
			FontSize:     req.FontSize,
		}
		err = h.gen.GenerateAIThumbnail(cfg, req.Output)

	case string(thumbnail.TemplateCentered):
		cfg := thumbnail.CenteredConfig{
			FramePath:   req.VideoPath,
			TextOverlay: req.Text,
			FontColor:   req.FontColor,
			FontSize:    req.FontSize,
		}
		err = h.gen.GenerateCentered(cfg, req.Output)

	default: // "branded" or empty → branded
		cfg := thumbnail.BrandedConfig{
			BackgroundPath: req.VideoPath,
			TextOverlay:    req.Text,
			FontColor:      req.FontColor,
			FontSize:       req.FontSize,
		}
		err = h.gen.GenerateBranded(cfg, req.Output)
	}

	if err != nil {
		h.log.Error("thumbnail generation failed", "err", err, "strategy", strategy)
		writeError(w, http.StatusInternalServerError, "generation_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"thumbnail_path": req.Output})
	if req.SessionID != "" {
		persistStepOutput(h.pipelineRepo, req.SessionID, "thumbnail", map[string]string{"thumbnail_path": req.Output})
	}
}
