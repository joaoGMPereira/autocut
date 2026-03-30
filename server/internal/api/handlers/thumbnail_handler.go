package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/thumbnail"
)

// ThumbnailHandler handles synchronous thumbnail generation (no SSE).
type ThumbnailHandler struct {
	gen *thumbnail.ThumbnailGenerator
	log *slog.Logger
}

// NewThumbnailHandler creates a ThumbnailHandler.
func NewThumbnailHandler(gen *thumbnail.ThumbnailGenerator) *ThumbnailHandler {
	return &ThumbnailHandler{
		gen: gen,
		log: slog.With("component", "api", "handler", "thumbnail"),
	}
}

type thumbnailRequest struct {
	VideoPath  string `json:"video_path"`
	Template   string `json:"template"`    // "branded" | "centered"
	Text       string `json:"text"`
	FontColor  string `json:"font_color"`
	FontSize   int    `json:"font_size"`
	Output     string `json:"output"`
}

// PostThumbnail handles POST /api/thumbnail (synchronous — no SSE).
func (h *ThumbnailHandler) PostThumbnail(w http.ResponseWriter, r *http.Request) {
	var req thumbnailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.VideoPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path is required")
		return
	}
	if req.Output == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "output is required")
		return
	}

	var err error
	switch req.Template {
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
		h.log.Error("thumbnail generation failed", "err", err, "template", req.Template)
		writeError(w, http.StatusInternalServerError, "generation_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"thumbnail_path": req.Output})
}
