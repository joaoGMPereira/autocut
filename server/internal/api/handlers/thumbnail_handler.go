package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/thumbnail"
)

// ThumbnailHandler handles HTTP requests for thumbnail generation and templates.
type ThumbnailHandler struct {
	runRepo     *database.PipelineRunRepo
	clipRepo    *database.PipelineClipRepo
	settingRepo *database.AppSettingRepo
	generator   *thumbnail.Generator
	hub         *hub.SSEHub
	log         *slog.Logger
}

// NewThumbnailHandler constructs a ThumbnailHandler.
func NewThumbnailHandler(
	runRepo *database.PipelineRunRepo,
	clipRepo *database.PipelineClipRepo,
	settingRepo *database.AppSettingRepo,
	h *hub.SSEHub,
	dataDir string,
) *ThumbnailHandler {
	gen := thumbnail.NewGenerator(clipRepo, h, dataDir)
	return &ThumbnailHandler{
		runRepo:     runRepo,
		clipRepo:    clipRepo,
		settingRepo: settingRepo,
		generator:   gen,
		hub:         h,
		log:         slog.With("handler", "thumbnail"),
	}
}

// PostBatchGenerate starts batch thumbnail generation for all clips in a run.
// POST /api/thumbnail/runs/{id}/generate
func (h *ThumbnailHandler) PostBatchGenerate(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}

	// Check ffmpeg availability
	if err := thumbnail.CheckFFmpegAvailable(); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate run state
	run, err := h.runRepo.GetByID(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, "run not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to load run", http.StatusInternalServerError)
		return
	}
	if run.State != "WAITING_THUMBNAIL_CONFIG" {
		jsonError(w, fmt.Sprintf("run not in WAITING_THUMBNAIL_CONFIG state (current: %s)", run.State), http.StatusConflict)
		return
	}

	// Parse request
	var req thumbnail.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := req.Config.Validate(); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Load clips
	clips, err := h.clipRepo.ListByRun(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to load clips", http.StatusInternalServerError)
		return
	}
	if len(clips) == 0 {
		jsonError(w, "no clips found for this run", http.StatusBadRequest)
		return
	}

	jobID := fmt.Sprintf("thumb_%d", runID)

	// Launch async generation
	go h.generator.GenerateBatch(context.Background(), runID, clips, req.Config, run.Mode, &req)

	// Return 202 immediately
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(thumbnail.BatchGenerateResponse{
		JobID:      jobID,
		TotalClips: len(clips),
	})
}

// PostSingleGenerate regenerates a single clip's thumbnail.
// POST /api/thumbnail/runs/{id}/clips/{clipId}/generate
func (h *ThumbnailHandler) PostSingleGenerate(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	clipIDStr := r.PathValue("clipId")
	clipID, err := strconv.ParseInt(clipIDStr, 10, 64)
	if err != nil || clipID <= 0 {
		jsonError(w, "invalid clip id", http.StatusBadRequest)
		return
	}

	// Check ffmpeg
	if err := thumbnail.CheckFFmpegAvailable(); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate run state
	run, err := h.runRepo.GetByID(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, "run not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to load run", http.StatusInternalServerError)
		return
	}
	if run.State != "WAITING_THUMBNAIL_CONFIG" {
		jsonError(w, fmt.Sprintf("run not in WAITING_THUMBNAIL_CONFIG state (current: %s)", run.State), http.StatusConflict)
		return
	}

	// Find the clip
	clips, err := h.clipRepo.ListByRun(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to load clips", http.StatusInternalServerError)
		return
	}
	var targetClip *database.PipelineClip
	for i := range clips {
		if clips[i].ID == clipID {
			targetClip = &clips[i]
			break
		}
	}
	if targetClip == nil {
		jsonError(w, "clip not found in this run", http.StatusNotFound)
		return
	}

	// Parse request
	var req thumbnail.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := req.Config.Validate(); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate synchronously
	thumbPath, err := h.generator.GenerateSingle(r.Context(), runID, *targetClip, req.Config, run.Mode, &req)
	if err != nil {
		h.log.Error("single thumbnail generation failed", "run_id", runID, "clip_id", clipID, "err", err)
		jsonError(w, fmt.Sprintf("generation failed: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(thumbnail.SingleGenerateResponse{
		ClipID:        clipID,
		ThumbnailPath: thumbPath,
	})
}

// GetFonts lists available system fonts.
// GET /api/thumbnail/fonts
func (h *ThumbnailHandler) GetFonts(w http.ResponseWriter, r *http.Request) {
	fonts := thumbnail.DetectFonts()
	defaultFont := thumbnail.DefaultFontFamily(fonts)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(thumbnail.FontListResponse{
		Fonts:       fonts,
		DefaultFont: defaultFont,
	})
}

// GetThumbnailFile serves the generated thumbnail image.
// GET /api/thumbnail/runs/{id}/clips/{clipId}/file
func (h *ThumbnailHandler) GetThumbnailFile(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	clipIDStr := r.PathValue("clipId")
	clipID, err := strconv.ParseInt(clipIDStr, 10, 64)
	if err != nil || clipID <= 0 {
		jsonError(w, "invalid clip id", http.StatusBadRequest)
		return
	}

	// Find the clip
	clips, err := h.clipRepo.ListByRun(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to load clips", http.StatusInternalServerError)
		return
	}
	var targetClip *database.PipelineClip
	for i := range clips {
		if clips[i].ID == clipID {
			targetClip = &clips[i]
			break
		}
	}
	if targetClip == nil || targetClip.ThumbnailPath == "" {
		jsonError(w, "thumbnail not generated", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(targetClip.ThumbnailPath); err != nil {
		jsonError(w, "thumbnail file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, targetClip.ThumbnailPath)
}

// GetTemplates lists all saved thumbnail templates.
// GET /api/thumbnail/templates
func (h *ThumbnailHandler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingRepo.List(r.Context())
	if err != nil {
		jsonError(w, "failed to list templates", http.StatusInternalServerError)
		return
	}

	var templates []thumbnail.ThumbnailTemplate
	for _, s := range settings {
		if !strings.HasPrefix(s.Key, "thumbnail_template_") {
			continue
		}
		var tmpl thumbnail.ThumbnailTemplate
		if err := json.Unmarshal([]byte(s.Value), &tmpl); err != nil {
			h.log.Warn("failed to parse template", "key", s.Key, "err", err)
			continue
		}
		templates = append(templates, tmpl)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"templates": templates})
}

// PostTemplate saves a new thumbnail template.
// POST /api/thumbnail/templates
func (h *ThumbnailHandler) PostTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string                  `json:"name"`
		Config thumbnail.ThumbnailConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		jsonError(w, "template name is required", http.StatusBadRequest)
		return
	}
	if err := req.Config.Validate(); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	key := "thumbnail_template_" + slugify(req.Name)

	// Check for duplicate
	existing, err := h.settingRepo.Get(r.Context(), key)
	if err != nil {
		jsonError(w, "failed to check template", http.StatusInternalServerError)
		return
	}
	if existing != "" {
		jsonError(w, "template with this name already exists", http.StatusConflict)
		return
	}

	tmpl := thumbnail.ThumbnailTemplate{
		Name:      req.Name,
		Config:    req.Config,
		CreatedAt: time.Now().UnixMilli(),
	}
	// Clear clip_texts from template config (per spec)
	tmpl.Config.ClipTexts = nil

	data, err := json.Marshal(tmpl)
	if err != nil {
		jsonError(w, "failed to serialize template", http.StatusInternalServerError)
		return
	}

	if err := h.settingRepo.Set(r.Context(), key, string(data)); err != nil {
		jsonError(w, "failed to save template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"name":       tmpl.Name,
		"created_at": tmpl.CreatedAt,
	})
}

// DeleteTemplate removes a saved thumbnail template.
// DELETE /api/thumbnail/templates/{name}
func (h *ThumbnailHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "template name is required", http.StatusBadRequest)
		return
	}

	key := "thumbnail_template_" + slugify(name)

	existing, err := h.settingRepo.Get(r.Context(), key)
	if err != nil {
		jsonError(w, "failed to check template", http.StatusInternalServerError)
		return
	}
	if existing == "" {
		jsonError(w, "template not found", http.StatusNotFound)
		return
	}

	if err := h.settingRepo.Delete(r.Context(), key); err != nil {
		jsonError(w, "failed to delete template", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// slugify converts a template name to a URL-safe key suffix.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "_")
	// Keep only alphanumeric and underscores
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		}
	}
	return b.String()
}
