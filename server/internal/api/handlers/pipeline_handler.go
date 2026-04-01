package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// PipelineHandler handles pipeline run CRUD.
type PipelineHandler struct {
	repo *database.PipelineRunRepo
	log  *slog.Logger
}

// NewPipelineHandler creates a PipelineHandler.
func NewPipelineHandler(repo *database.PipelineRunRepo) *PipelineHandler {
	return &PipelineHandler{
		repo: repo,
		log:  slog.With("component", "api", "handler", "pipeline"),
	}
}

type postRunRequest struct {
	URL       string          `json:"url"`
	Mode      string          `json:"mode"`
	ChannelID *int64          `json:"channel_id,omitempty"`
	Config    json.RawMessage `json:"config,omitempty"`
}

// PostRun handles POST /api/pipeline/runs.
func (h *PipelineHandler) PostRun(w http.ResponseWriter, r *http.Request) {
	var req postRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "url is required")
		return
	}

	modeConfigJSON := "{}"
	if len(req.Config) > 0 && string(req.Config) != "null" {
		modeConfigJSON = string(req.Config)
	}

	run := &database.PipelineRun{
		URL:            req.URL,
		Mode:           req.Mode,
		Status:         "pending",
		ModeConfigJSON: modeConfigJSON,
	}
	if req.ChannelID != nil {
		run.ChannelID = sql.NullInt64{Int64: *req.ChannelID, Valid: true}
	}

	id, err := h.repo.Create(r.Context(), run)
	if err != nil {
		h.log.Error("create pipeline run", "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"run_id": id})
}

type patchRunModeRequest struct {
	Mode   string          `json:"mode"`
	Config json.RawMessage `json:"config,omitempty"`
}

// PatchRunMode handles PATCH /api/pipeline/runs/{id}/mode.
func (h *PipelineHandler) PatchRunMode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be an integer")
		return
	}

	var req patchRunModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	modeConfigJSON := "{}"
	if len(req.Config) > 0 && string(req.Config) != "null" {
		modeConfigJSON = string(req.Config)
	}

	if err := h.repo.UpdateMode(r.Context(), id, req.Mode, modeConfigJSON); err != nil {
		h.log.Error("update pipeline run mode", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetRun handles GET /api/pipeline/runs/{id}.
func (h *PipelineHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be an integer")
		return
	}

	run, steps, err := h.repo.GetWithSteps(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "not_found", "pipeline run not found")
			return
		}
		h.log.Error("get pipeline run", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	// Parse step_outputs_json into a map for the response
	var stepOutputs map[string]interface{}
	if err := json.Unmarshal([]byte(run.StepOutputsJSON), &stepOutputs); err != nil {
		stepOutputs = map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"run":          run,
		"step_outputs": stepOutputs,
		"steps":        steps,
	})
}

// ListRuns handles GET /api/pipeline/runs.
func (h *PipelineHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	offset := 0
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

	runs, total, err := h.repo.ListAll(r.Context(), limit, offset, channelID)
	if err != nil {
		h.log.Error("list pipeline runs", "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	if runs == nil {
		runs = []database.PipelineRun{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"runs":  runs,
		"total": total,
	})
}

// DeleteRun handles DELETE /api/pipeline/runs/{id}.
func (h *PipelineHandler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be an integer")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "not_found", "pipeline run not found")
			return
		}
		h.log.Error("delete pipeline run", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// stepOutputs is a helper type for deserialising StepOutputsJSON.
type stepOutputs struct {
	Download struct {
		FilePath string `json:"file_path"`
	} `json:"download"`
}

// previewSpeedConfig mirrors the frontend SpeedConfig for preview filter generation.
type previewSpeedConfig struct {
	Enabled bool    `json:"enabled"`
	Factor  float64 `json:"factor"`
}

// previewVisualConfig mirrors the frontend VisualConfig for preview filter generation.
type previewVisualConfig struct {
	CropEnabled  bool    `json:"cropEnabled"`
	CropPercent  float64 `json:"cropPercent"`
	ZoomEnabled  bool    `json:"zoomEnabled"`
	ZoomAmount   float64 `json:"zoomAmount"`
	ColorGrading bool    `json:"colorGrading"`
	Brightness   float64 `json:"brightness"`
	Saturation   float64 `json:"saturation"`
	Contrast     float64 `json:"contrast"`
}

// previewModeConfig holds the fields from mode_config_json needed for preview filter generation.
type previewModeConfig struct {
	AntiDuplicate     bool                `json:"antiDuplicate"`
	AntiDuplicateMode string              `json:"antiDuplicateMode"` // "subtle" | "aggressive"
	Speed             previewSpeedConfig  `json:"speed"`
	Visual            previewVisualConfig `json:"visual"`
}

// buildPreviewFilters returns ffmpeg -vf and -af filter strings based on the mode config.
// Returns empty strings when no anti-duplication is configured (fast -c copy path).
func buildPreviewFilters(cfg previewModeConfig) (vf, af string) {
	if !cfg.AntiDuplicate {
		return
	}

	// Legacy format: old ModeConfig without nested Speed/Visual sub-configs.
	// Detected when Speed.Factor == 0 and Visual.CropPercent == 0.
	isLegacy := cfg.Speed.Factor == 0 && cfg.Visual.CropPercent == 0

	var (
		speedFactor     float64
		cropPct         float64
		zoomAmt         float64
		applyColor      bool
		colorBrightness float64
		colorSaturation float64
		colorContrast   float64
	)

	if isLegacy {
		switch cfg.AntiDuplicateMode {
		case "aggressive":
			speedFactor, cropPct = 1.05, 3.0
			applyColor, colorBrightness, colorSaturation, colorContrast = true, 0.03, 1.05, 1.02
		default: // subtle
			speedFactor, cropPct = 1.02, 2.0
		}
	} else {
		if cfg.Speed.Enabled && cfg.Speed.Factor > 1.0 {
			speedFactor = cfg.Speed.Factor
		}
		if cfg.Visual.CropEnabled && cfg.Visual.CropPercent > 0 {
			cropPct = cfg.Visual.CropPercent
		}
		if cfg.Visual.ZoomEnabled && cfg.Visual.ZoomAmount > 1.0 {
			zoomAmt = cfg.Visual.ZoomAmount
		}
		if cfg.Visual.ColorGrading {
			applyColor = true
			colorBrightness = cfg.Visual.Brightness
			colorSaturation = cfg.Visual.Saturation
			if colorSaturation == 0 {
				colorSaturation = 1.0
			}
			colorContrast = cfg.Visual.Contrast
			if colorContrast == 0 {
				colorContrast = 1.0
			}
		}
	}

	var vFilters []string

	// crop=w:h:x:y — keep center (1-cropPct/100) of frame
	if cropPct > 0 {
		f := 1.0 - cropPct/100.0
		half := cropPct / 200.0
		vFilters = append(vFilters, fmt.Sprintf(
			"crop=iw*%.4f:ih*%.4f:iw*%.4f:ih*%.4f", f, f, half, half))
	}

	// Zoom: scale up then crop back to preserve output dimensions
	if zoomAmt > 1.0 {
		z := zoomAmt
		vFilters = append(vFilters, fmt.Sprintf("scale=iw*%.4f:ih*%.4f", z, z))
		vFilters = append(vFilters, fmt.Sprintf(
			"crop=iw/%.4f:ih/%.4f:(iw-iw/%.4f)/2:(ih-ih/%.4f)/2", z, z, z, z))
	}

	// Color grading via eq filter
	if applyColor {
		vFilters = append(vFilters, fmt.Sprintf(
			"eq=brightness=%.4f:saturation=%.4f:contrast=%.4f",
			colorBrightness, colorSaturation, colorContrast))
	}

	// Speed: setpts for video, atempo for audio
	if speedFactor > 1.0 {
		vFilters = append(vFilters, fmt.Sprintf("setpts=PTS/%.4f", speedFactor))
		af = fmt.Sprintf("atempo=%.4f", speedFactor)
	}

	vf = strings.Join(vFilters, ",")
	return
}

// autoCutDir returns the AutoCut working directory, honouring the AUTOCUT_DIR
// environment variable and falling back to ~/.autocut.
func autoCutDir() (string, error) {
	if d := os.Getenv("AUTOCUT_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".autocut"), nil
}

// PostRunPreview handles POST /api/pipeline/runs/{id}/preview.
// It extracts the first 30 seconds of the downloaded source video and saves it
// as a preview under ~/.autocut/previews/{id}.mp4.
func (h *PipelineHandler) PostRunPreview(w http.ResponseWriter, r *http.Request) {
	// 1. Parse {id}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be an integer")
		return
	}

	// 2. Load run via h.repo.GetByID
	run, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "not_found", "pipeline run not found")
			return
		}
		h.log.Error("get pipeline run for preview", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	// 3. Parse StepOutputsJSON → find download.file_path
	var outputs stepOutputs
	if run.StepOutputsJSON != "" {
		if err := json.Unmarshal([]byte(run.StepOutputsJSON), &outputs); err != nil {
			h.log.Warn("parse step_outputs_json", "id", id, "err", err)
		}
	}

	// 4. If no file_path → 409 "video_not_ready"
	if outputs.Download.FilePath == "" {
		writeError(w, http.StatusConflict, "video_not_ready", "download step has not completed yet")
		return
	}

	// 5. If file_path doesn't exist on disk → 409 "file_not_found"
	if _, err := os.Stat(outputs.Download.FilePath); os.IsNotExist(err) {
		writeError(w, http.StatusConflict, "file_not_found", "source video file not found on disk")
		return
	}

	// Parse mode config to apply anti-duplication effects in the preview
	var modeCfg previewModeConfig
	if run.ModeConfigJSON != "" && run.ModeConfigJSON != "{}" {
		if err := json.Unmarshal([]byte(run.ModeConfigJSON), &modeCfg); err != nil {
			h.log.Warn("parse mode_config_json for preview", "id", id, "err", err)
		}
	}
	previewVF, previewAF := buildPreviewFilters(modeCfg)

	// 6. Create preview dir: ~/.autocut/previews/
	baseDir, err := autoCutDir()
	if err != nil {
		h.log.Error("resolve autocut dir", "err", err)
		writeError(w, http.StatusInternalServerError, "config_error", err.Error())
		return
	}
	previewDir := filepath.Join(baseDir, "previews")
	if err := os.MkdirAll(previewDir, 0o755); err != nil {
		h.log.Error("create previews dir", "dir", previewDir, "err", err)
		writeError(w, http.StatusInternalServerError, "fs_error", err.Error())
		return
	}

	// 7. Run ffmpeg with optional anti-duplication filters.
	// When filters are active: re-encode (120s timeout). Without: fast -c copy (60s).
	dst := filepath.Join(previewDir, fmt.Sprintf("%d.mp4", id))
	timeout := 60 * time.Second
	if previewVF != "" || previewAF != "" {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	args := []string{"-y", "-i", outputs.Download.FilePath, "-t", "30"}
	if previewVF != "" {
		args = append(args, "-vf", previewVF)
	}
	if previewAF != "" {
		args = append(args, "-af", previewAF)
	} else if previewVF != "" {
		// Visual-only filters: copy audio for performance
		args = append(args, "-c:a", "copy")
	}
	if previewVF == "" && previewAF == "" {
		// No filters: fast stream copy
		args = append(args, "-c", "copy")
	}
	args = append(args, dst)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		h.log.Error("ffmpeg preview", "id", id, "err", err, "output", string(out))
		writeError(w, http.StatusInternalServerError, "ffmpeg_error", "failed to generate preview")
		return
	}

	// 8. Return JSON: {"preview_url": "/files/previews/{id}.mp4"}
	writeJSON(w, http.StatusOK, map[string]string{
		"preview_url": fmt.Sprintf("/files/previews/%d.mp4", id),
	})
}
