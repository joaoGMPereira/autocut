package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/processor"
)

// PreviewHandler handles pipeline run preview generation requests.
type PreviewHandler struct {
	runRepo        *database.PipelineRunRepo
	channelCfgRepo *database.ChannelConfigRepo
	settingRepo    *database.AppSettingRepo
	dataDir        string
	hub            *hub.SSEHub
	log            *slog.Logger
	inFlight       sync.Map // map[int64]chan struct{} — per-run concurrency guard
}

// NewPreviewHandler creates a PreviewHandler with all dependencies.
func NewPreviewHandler(
	runRepo *database.PipelineRunRepo,
	channelCfgRepo *database.ChannelConfigRepo,
	settingRepo *database.AppSettingRepo,
	dataDir string,
	h *hub.SSEHub,
) *PreviewHandler {
	return &PreviewHandler{
		runRepo:        runRepo,
		channelCfgRepo: channelCfgRepo,
		settingRepo:    settingRepo,
		dataDir:        dataDir,
		hub:            h,
		log:            slog.With("handler", "preview"),
	}
}

// postPreviewRequest is the JSON body for POST /preview.
type postPreviewRequest struct {
	ModeConfig json.RawMessage `json:"mode_config"`
}

// PostPreview handles POST /api/pipeline/runs/{id}/preview.
// Starts preview generation asynchronously, returns 202 Accepted.
func (h *PreviewHandler) PostPreview(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req postPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ModeConfig == nil {
		req.ModeConfig = json.RawMessage("{}")
	}

	// Look up the run
	run, err := h.runRepo.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("run lookup failed", "id", id, "err", err)
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	// Validate video_path exists on disk
	if run.VideoPath == "" {
		jsonError(w, "source video not found", http.StatusUnprocessableEntity)
		return
	}
	if _, err := os.Stat(run.VideoPath); os.IsNotExist(err) {
		jsonError(w, "source video not found", http.StatusUnprocessableEntity)
		return
	}

	// Check ffmpeg availability
	if !processor.FFmpegAvailable() {
		jsonError(w, "ffmpeg not available", http.StatusServiceUnavailable)
		return
	}

	// Concurrency guard — prevent duplicate ffmpeg spawns for same run
	if _, loaded := h.inFlight.LoadOrStore(id, make(chan struct{})); loaded {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": id, "status": "pending"})
		return
	}

	// Load channel config
	var channelCfg database.ChannelConfig
	if run.ChannelID.Valid {
		cc, err := h.channelCfgRepo.GetByChannelID(r.Context(), run.ChannelID.Int64)
		if err != nil {
			h.log.Warn("channel config lookup failed, using defaults", "channel_id", run.ChannelID.Int64, "err", err)
		} else {
			channelCfg = *cc
		}
	}

	// Apply mode overrides from the frontend ModeConfig
	processor.ApplyModeOverrides(&channelCfg, req.ModeConfig)
	blurEdgePct, noiseStrength := processor.ParseAntiDupExtras(req.ModeConfig)

	// Resolve relative watermark path: try overlay_library_path from settings, then dataDir
	overlayLibraryPath, _ := h.settingRepo.Get(r.Context(), "overlay_library_path")
	resolveVideoOverlayPath(&channelCfg, overlayLibraryPath, h.dataDir)

	// Fallback: use channel avatar as logo when logo is enabled but path is missing
	if channelCfg.BrandingLogoEnabled && (!channelCfg.BrandingLogoPath.Valid || channelCfg.BrandingLogoPath.String == "") && run.ChannelID.Valid {
		cid := run.ChannelID.Int64
		for _, ext := range []string{".jpg", ".jpeg", ".png"} {
			avatarPath := filepath.Join(h.dataDir, "avatars", fmt.Sprintf("%d%s", cid, ext))
			if _, err := os.Stat(avatarPath); err == nil {
				channelCfg.BrandingLogoPath = sql.NullString{String: avatarPath, Valid: true}
				h.log.Info("logo fallback: using channel avatar", "path", avatarPath)
				break
			}
		}
	}

	// Resolve transcript path: use run's own path, or look up prior run with same URL
	transcriptPath := run.TranscriptPath
	if transcriptPath == "" && run.URL != "" {
		if found, err := h.runRepo.FindTranscriptByURL(r.Context(), run.URL); err == nil && found != "" {
			transcriptPath = found
			h.log.Info("transcript fallback: using prior run", "url", run.URL, "path", found)
		}
	}

	// Return 202 immediately, spawn generation goroutine
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"run_id": id, "status": "pending"})

	jobID := fmt.Sprintf("%d", id)

	go func() {
		defer h.inFlight.Delete(id)

		previewReq := processor.PreviewRequest{
			RunID:          id,
			VideoPath:      run.VideoPath,
			DurationSec:    run.DurationSec,
			ChannelCfg:     channelCfg,
			ModeCfg:        req.ModeConfig,
			DataDir:        h.dataDir,
			BlurEdgePct:    blurEdgePct,
			NoiseStrength:  noiseStrength,
			TranscriptPath: transcriptPath,
			OnProgress: func(pct float64) {
				h.hub.Publish(jobID, hub.SSEEvent{
					Type: "preview_progress",
					Data: map[string]any{"run_id": id, "percent": pct},
				})
			},
		}

		outPath, err := processor.GeneratePreview(context.Background(), previewReq)
		if err != nil {
			h.log.Error("preview generation failed", "run_id", id, "err", err)
			h.hub.Publish(jobID, hub.SSEEvent{
				Type: "preview_error",
				Data: map[string]any{"run_id": id, "message": err.Error()},
			})
			return
		}

		_ = outPath // path used by GET endpoint
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "preview_ready",
			Data: map[string]any{
				"run_id": id,
				"url":    fmt.Sprintf("/api/pipeline/runs/%d/preview/file", id),
			},
		})
	}()
}

// GetPreviewFile handles GET /api/pipeline/runs/{id}/preview/file.
// Serves the generated preview MP4 with Range/206 support.
func (h *PreviewHandler) GetPreviewFile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	// Find preview file by glob pattern
	previewDir := filepath.Join(h.dataDir, "previews")
	pattern := filepath.Join(previewDir, fmt.Sprintf("%d_*.mp4", id))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}

	// Use the most recent match (in case of multiple cache entries)
	previewPath := matches[len(matches)-1]

	f, err := os.Open(previewPath)
	if err != nil {
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "video/mp4")
	// http.ServeContent handles Range headers, ETag, and 206 automatically
	http.ServeContent(w, r, filepath.Base(previewPath), stat.ModTime(), f)
}

// resolveVideoOverlayPath patches cc.PreviewVideoOverlayConfigJSON so that relative
// video paths are resolved against dataDir. This handles the case where the frontend
// stores just a filename (e.g. "seguir-1.mp4") without a full path.
// Search order: overlayLibraryPath, then dataDir.
func resolveVideoOverlayPath(cc *database.ChannelConfig, overlayLibraryPath, dataDir string) {
	if cc.PreviewVideoOverlayConfigJSON == "" || cc.PreviewVideoOverlayConfigJSON == "{}" {
		return
	}
	var cfg processor.VideoOverlayConfig
	if err := json.Unmarshal([]byte(cc.PreviewVideoOverlayConfigJSON), &cfg); err != nil {
		return
	}
	if !cfg.Enabled || cfg.Path == "" {
		return
	}
	if filepath.IsAbs(cfg.Path) {
		return
	}
	// Try overlay library path first, then dataDir
	searchDirs := []string{overlayLibraryPath, dataDir}
	for _, dir := range searchDirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, cfg.Path)
		if _, err := os.Stat(candidate); err == nil {
			cfg.Path = candidate
			if b, err := json.Marshal(cfg); err == nil {
				cc.PreviewVideoOverlayConfigJSON = string(b)
			}
			return
		}
	}
}

