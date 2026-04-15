package processor

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// PreviewRequest contains everything needed to generate a preview video.
type PreviewRequest struct {
	RunID       int64
	VideoPath   string
	DurationSec int64
	ChannelCfg  database.ChannelConfig
	ModeCfg     json.RawMessage // raw ModeConfig JSON from POST body
	DataDir     string          // root dir for output files ({dataDir}/previews/)
	OnProgress  func(float64)   // callback: percent 0–100
}

// VideoOverlayConfig is parsed from PreviewVideoOverlayConfigJSON.
type VideoOverlayConfig struct {
	Enabled   bool   `json:"enabled"`
	Path      string `json:"path"`
	ScalePct  int    `json:"scale_pct"`
	Position  string `json:"position"`
	ChromaKey string `json:"chroma_key"`
	Appearances []struct {
		StartPct float64 `json:"start_pct"`
		EndPct   float64 `json:"end_pct"`
	} `json:"appearances"`
}

// chromaKeyParams maps color names to colorkey filter parameters.
var chromaKeyParams = map[string][3]string{
	"green": {"0x00FF00", "0.1", "0.1"},
	"verde": {"0x00FF00", "0.1", "0.1"},
	"black": {"0x000000", "0.3", "0.15"},
	"preto": {"0x000000", "0.3", "0.15"},
	"white": {"0xFFFFFF", "0.25", "0.1"},
	"branco": {"0xFFFFFF", "0.25", "0.1"},
	"blue":  {"0x0000FF", "0.1", "0.1"},
	"azul":  {"0x0000FF", "0.1", "0.1"},
}

// GeneratePreview produces a preview MP4 for a pipeline run.
// Returns the path to the output file on success.
func GeneratePreview(ctx context.Context, req PreviewRequest) (string, error) {
	log := slog.With("component", "preview", "run_id", req.RunID)

	hash, err := computeCacheHash(req.ChannelCfg, req.ModeCfg)
	if err != nil {
		return "", fmt.Errorf("compute cache hash: %w", err)
	}

	previewDir := filepath.Join(req.DataDir, "previews")
	if err := os.MkdirAll(previewDir, 0o755); err != nil {
		return "", fmt.Errorf("create previews dir: %w", err)
	}

	outPath := filepath.Join(previewDir, fmt.Sprintf("%d_%s.mp4", req.RunID, hash))

	// Cache hit
	if _, err := os.Stat(outPath); err == nil {
		log.Info("preview cache hit", "path", outPath)
		return outPath, nil
	}

	// Smart start position: ~20% into the video to skip intros/credits
	startSec := max(10, int(float64(req.DurationSec)*0.20))
	if int64(startSec) >= req.DurationSec {
		startSec = 0 // fallback for very short videos
	}
	segmentDuration := min(int64(120), req.DurationSec-int64(startSec))
	if segmentDuration <= 0 {
		segmentDuration = req.DurationSec
		startSec = 0
	}

	// Build args with all applicable layers
	args, cleanups := buildPreviewArgs(log, req, startSec, segmentDuration, outPath)
	for _, fn := range cleanups {
		defer fn()
	}

	log.Info("generating preview", "output", outPath, "start_sec", startSec, "duration", segmentDuration)

	err = RunFFmpeg(ctx, args, float64(segmentDuration), req.OnProgress)
	if err != nil {
		_ = os.Remove(outPath)
		return "", fmt.Errorf("ffmpeg preview: %w", err)
	}

	log.Info("preview generated", "path", outPath)
	return outPath, nil
}

// buildPreviewArgs constructs the full ffmpeg command including all effect layers.
// Returns the args slice and any cleanup functions (e.g., for temp ASS files).
func buildPreviewArgs(log *slog.Logger, req PreviewRequest, startSec int, duration int64, outPath string) ([]string, []func()) {
	cc := req.ChannelCfg
	var cleanups []func()

	// Collect video filters and extra inputs
	var vFilters []string
	var extraInputs []string
	inputIdx := 1 // input 0 is the main video

	// T010: Color grading
	if cc.VisualColorGradingEnabled {
		eq := fmt.Sprintf("eq=brightness=%.4f:saturation=%.4f:contrast=%.4f",
			cc.VisualBrightness, cc.VisualSaturation, cc.VisualContrast)
		vFilters = append(vFilters, eq)
		log.Info("preview layer: color grading")
	} else {
		log.Warn("preview layer skipped: color grading (disabled)")
	}

	// T011: Crop
	if cc.VisualCropEnabled && cc.VisualCropPercent > 0 {
		factor := 1.0 - cc.VisualCropPercent
		crop := fmt.Sprintf("crop=iw*%.4f:ih*%.4f", factor, factor)
		vFilters = append(vFilters, crop)
		log.Info("preview layer: crop", "percent", cc.VisualCropPercent)
	} else {
		log.Warn("preview layer skipped: crop (disabled)")
	}

	// T012: Logo overlay
	if cc.BrandingLogoEnabled && cc.BrandingLogoPath.Valid && cc.BrandingLogoPath.String != "" {
		logoPath := cc.BrandingLogoPath.String
		if _, err := os.Stat(logoPath); err == nil {
			extraInputs = append(extraInputs, "-i", logoPath)
			// Calculate overlay position and opacity
			opacity := cc.BrandingLogoOpacity
			scale := cc.BrandingLogoScale
			pos := overlayPosition(cc.BrandingLogoPosition, scale)
			overlay := fmt.Sprintf("[%d]format=rgba,colorchannelmixer=aa=%.2f,scale=iw*%.2f:ih*%.2f[logo];[prev][logo]overlay=%s",
				inputIdx, opacity, scale, scale, pos)
			// We need to chain filters using named outputs
			// Actually, let's use a simpler approach with filter_complex
			_ = overlay
			inputIdx++
			// For logo, we'll handle it in the filter_complex
			vFilters = append(vFilters, fmt.Sprintf("LOGO_INPUT:%d:%.2f:%.2f:%s", inputIdx-1, opacity, scale, cc.BrandingLogoPosition))
			log.Info("preview layer: logo overlay", "path", logoPath)
		} else {
			log.Warn("preview layer skipped: logo (file not found)", "path", logoPath)
		}
	} else {
		log.Warn("preview layer skipped: logo (disabled)")
	}

	// T013: Watermark video overlay
	var overlayConfig VideoOverlayConfig
	if cc.PreviewVideoOverlayConfigJSON != "" && cc.PreviewVideoOverlayConfigJSON != "{}" {
		if err := json.Unmarshal([]byte(cc.PreviewVideoOverlayConfigJSON), &overlayConfig); err != nil {
			log.Warn("preview layer skipped: video overlay (invalid JSON)", "err", err)
		}
	}
	if overlayConfig.Enabled && overlayConfig.Path != "" {
		if _, err := os.Stat(overlayConfig.Path); err == nil {
			extraInputs = append(extraInputs, "-i", overlayConfig.Path)
			vFilters = append(vFilters, fmt.Sprintf("WATERMARK_INPUT:%d:%s:%d:%s",
				inputIdx, overlayConfig.ChromaKey, overlayConfig.ScalePct, overlayConfig.Position))
			inputIdx++
			log.Info("preview layer: watermark overlay", "path", overlayConfig.Path)
		} else {
			log.Warn("preview layer skipped: watermark (file not found)", "path", overlayConfig.Path)
		}
	} else {
		log.Warn("preview layer skipped: watermark (disabled)")
	}

	// T014: Captions ASS burn-in
	if cc.PreviewCaptionsEnabled {
		var captionStyle CaptionTextStyle
		if cc.PreviewCaptionTextStyleJSON != "" && cc.PreviewCaptionTextStyleJSON != "{}" {
			if err := json.Unmarshal([]byte(cc.PreviewCaptionTextStyleJSON), &captionStyle); err != nil {
				log.Warn("caption style JSON parse failed, using defaults", "err", err)
			}
		}
		assPath, cleanup, err := GenerateASS(captionStyle)
		if err != nil {
			log.Warn("preview layer skipped: captions (ASS generation failed)", "err", err)
		} else {
			cleanups = append(cleanups, cleanup)
			// Escape colons and backslashes in path for ffmpeg filter
			escapedPath := strings.ReplaceAll(assPath, "\\", "\\\\")
			escapedPath = strings.ReplaceAll(escapedPath, ":", "\\:")
			vFilters = append(vFilters, fmt.Sprintf("subtitles=%s", escapedPath))
			log.Info("preview layer: captions", "ass_path", assPath)
		}
	} else {
		log.Warn("preview layer skipped: captions (disabled)")
	}

	// Now build the actual ffmpeg command
	// Check if we need filter_complex (for multi-input overlays) or simple -vf
	hasOverlayInputs := false
	var simpleFilters []string
	var complexParts []string
	logoInputIdx := -1
	watermarkInputIdx := -1

	for _, f := range vFilters {
		if strings.HasPrefix(f, "LOGO_INPUT:") {
			hasOverlayInputs = true
			parts := strings.SplitN(f, ":", 5)
			idx := 0
			fmt.Sscanf(parts[1], "%d", &idx)
			logoInputIdx = idx
		} else if strings.HasPrefix(f, "WATERMARK_INPUT:") {
			hasOverlayInputs = true
			parts := strings.SplitN(f, ":", 5)
			idx := 0
			fmt.Sscanf(parts[1], "%d", &idx)
			watermarkInputIdx = idx
		} else {
			simpleFilters = append(simpleFilters, f)
		}
	}

	if !hasOverlayInputs {
		// Simple case: only -vf filters (eq, crop, subtitles)
		args := []string{"-y", "-ss", fmt.Sprintf("%d", startSec), "-i", req.VideoPath, "-t", fmt.Sprintf("%d", duration)}
		if len(simpleFilters) > 0 {
			args = append(args, "-vf", strings.Join(simpleFilters, ","))
		}
		args = append(args, "-c:v", "libx264", "-preset", "fast", "-crf", "23",
			"-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", outPath)
		return args, cleanups
	}

	// Complex case: need -filter_complex for overlay inputs
	args := []string{"-y", "-ss", fmt.Sprintf("%d", startSec), "-i", req.VideoPath}
	args = append(args, extraInputs...)
	args = append(args, "-t", fmt.Sprintf("%d", duration))

	// Build filter_complex chain
	currentLabel := "[0:v]"

	// Apply simple filters first
	if len(simpleFilters) > 0 {
		complexParts = append(complexParts, fmt.Sprintf("%s%s[base]", currentLabel, strings.Join(simpleFilters, ",")))
		currentLabel = "[base]"
	}

	// Logo overlay
	if logoInputIdx >= 0 {
		// Find the original LOGO_INPUT filter spec
		for _, f := range vFilters {
			if strings.HasPrefix(f, "LOGO_INPUT:") {
				parts := strings.SplitN(f, ":", 5)
				opacity := 0.3
				scale := 0.1
				position := "BOTTOM_RIGHT"
				fmt.Sscanf(parts[2], "%f", &opacity)
				fmt.Sscanf(parts[3], "%f", &scale)
				if len(parts) > 4 {
					position = parts[4]
				}
				outLabel := "[afterlogo]"
				pos := overlayPosition(position, scale)
				complexParts = append(complexParts,
					fmt.Sprintf("[%d]format=rgba,colorchannelmixer=aa=%.2f,scale=iw*%.2f:-1[logo_scaled]", logoInputIdx, opacity, scale),
					fmt.Sprintf("%s[logo_scaled]overlay=%s%s", currentLabel, pos, outLabel),
				)
				currentLabel = outLabel
				break
			}
		}
	}

	// Watermark overlay
	if watermarkInputIdx >= 0 {
		for _, f := range vFilters {
			if strings.HasPrefix(f, "WATERMARK_INPUT:") {
				parts := strings.SplitN(f, ":", 5)
				chromaKey := ""
				scalePct := 100
				position := "center"
				if len(parts) > 2 {
					chromaKey = parts[2]
				}
				if len(parts) > 3 {
					fmt.Sscanf(parts[3], "%d", &scalePct)
				}
				if len(parts) > 4 {
					position = parts[4]
				}

				outLabel := "[afterwatermark]"
				scaleFilter := fmt.Sprintf("scale=iw*%d/100:-1", scalePct)

				if params, ok := chromaKeyParams[strings.ToLower(chromaKey)]; ok && strings.ToLower(chromaKey) != "none" {
					complexParts = append(complexParts,
						fmt.Sprintf("[%d]%s,colorkey=%s:%s:%s,format=rgba[wm_keyed]",
							watermarkInputIdx, scaleFilter, params[0], params[1], params[2]),
					)
					pos := overlayPosition(position, float64(scalePct)/100.0)
					complexParts = append(complexParts,
						fmt.Sprintf("%s[wm_keyed]overlay=%s:eof_action=pass%s", currentLabel, pos, outLabel),
					)
				} else {
					complexParts = append(complexParts,
						fmt.Sprintf("[%d]%s,format=rgba[wm_scaled]", watermarkInputIdx, scaleFilter),
					)
					pos := overlayPosition(position, float64(scalePct)/100.0)
					complexParts = append(complexParts,
						fmt.Sprintf("%s[wm_scaled]overlay=%s:eof_action=pass%s", currentLabel, pos, outLabel),
					)
				}
				currentLabel = outLabel
				break
			}
		}
	}

	// Map current label to output
	if currentLabel != "[0:v]" && currentLabel[0] == '[' {
		// The last filter needs to produce the output for -map
		args = append(args, "-filter_complex", strings.Join(complexParts, ";"))
		// Strip brackets from currentLabel for -map
		mapLabel := currentLabel
		args = append(args, "-map", mapLabel, "-map", "0:a?")
	}

	args = append(args, "-c:v", "libx264", "-preset", "fast", "-crf", "23",
		"-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", outPath)

	return args, cleanups
}

// overlayPosition maps a named position to ffmpeg overlay coordinates.
func overlayPosition(position string, scale float64) string {
	pos := strings.ToLower(strings.ReplaceAll(position, "_", ""))
	switch pos {
	case "topleft", "top_left":
		return "10:10"
	case "topcenter", "top_center":
		return "(W-w)/2:10"
	case "topright", "top_right":
		return "W-w-10:10"
	case "midleft", "mid_left", "middleleft":
		return "10:(H-h)/2"
	case "midcenter", "mid_center", "center", "middlecenter":
		return "(W-w)/2:(H-h)/2"
	case "midright", "mid_right", "middleright":
		return "W-w-10:(H-h)/2"
	case "bottomleft", "bottom_left":
		return "10:H-h-10"
	case "bottomcenter", "bottom_center":
		return "(W-w)/2:H-h-10"
	case "bottomright", "bottom_right":
		return "W-w-10:H-h-10"
	default:
		return "W-w-10:H-h-10" // default bottom-right
	}
}

// computeCacheHash returns MD5 hex of ChannelConfig + ModeConfig JSON.
func computeCacheHash(cc database.ChannelConfig, modeCfg json.RawMessage) (string, error) {
	ccJSON, err := json.Marshal(cc)
	if err != nil {
		return "", err
	}
	combined := append(ccJSON, modeCfg...)
	return fmt.Sprintf("%x", md5.Sum(combined)), nil
}
