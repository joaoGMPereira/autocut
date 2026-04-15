package processor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// EffectRequest holds the common config for any video processing job (preview or clip).
type EffectRequest struct {
	VideoPath   string
	StartSec    int
	DurationSec int64
	ChannelCfg  database.ChannelConfig
	OutputPath  string
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

// chromaKeyParams maps color names to colorkey filter parameters: {color, similarity, blend}.
var chromaKeyParams = map[string][3]string{
	"green":  {"0x00FF00", "0.1", "0.1"},
	"verde":  {"0x00FF00", "0.1", "0.1"},
	"black":  {"0x000000", "0.3", "0.15"},
	"preto":  {"0x000000", "0.3", "0.15"},
	"white":  {"0xFFFFFF", "0.25", "0.1"},
	"branco": {"0xFFFFFF", "0.25", "0.1"},
	"blue":   {"0x0000FF", "0.1", "0.1"},
	"azul":   {"0x0000FF", "0.1", "0.1"},
}

// BuildEffectChain constructs the full ffmpeg command with all applicable effect layers.
// Used by both preview and clip generation.
// Returns the args slice and any cleanup functions (e.g., for temp ASS files).
func BuildEffectChain(log *slog.Logger, req EffectRequest) ([]string, []func()) {
	cc := req.ChannelCfg
	var cleanups []func()

	// Collect simple video filters and extra inputs with metadata
	var simpleFilters []string
	var extraInputs []string
	var overlayOps []overlayOp // structured overlay operations
	inputIdx := 1              // input 0 is the main video

	// --- Layer 1: Color grading ---
	if cc.VisualColorGradingEnabled {
		eq := fmt.Sprintf("eq=brightness=%.4f:saturation=%.4f:contrast=%.4f",
			cc.VisualBrightness, cc.VisualSaturation, cc.VisualContrast)
		simpleFilters = append(simpleFilters, eq)
		log.Info("effect layer: color grading")
	}

	// --- Layer 2: Crop ---
	if cc.VisualCropEnabled && cc.VisualCropPercent > 0 {
		factor := 1.0 - cc.VisualCropPercent
		crop := fmt.Sprintf("crop=iw*%.4f:ih*%.4f", factor, factor)
		simpleFilters = append(simpleFilters, crop)
		log.Info("effect layer: crop", "percent", cc.VisualCropPercent)
	}

	// --- Layer 3: Logo overlay ---
	if cc.BrandingLogoEnabled && cc.BrandingLogoPath.Valid && cc.BrandingLogoPath.String != "" {
		logoPath := cc.BrandingLogoPath.String
		if _, err := os.Stat(logoPath); err == nil {
			extraInputs = append(extraInputs, "-i", logoPath)
			overlayOps = append(overlayOps, overlayOp{
				kind:     opLogo,
				inputIdx: inputIdx,
				opacity:  cc.BrandingLogoOpacity,
				scale:    cc.BrandingLogoScale,
				position: cc.BrandingLogoPosition,
			})
			inputIdx++
			log.Info("effect layer: logo overlay", "path", logoPath)
		} else {
			log.Warn("effect layer skipped: logo (file not found)", "path", logoPath)
		}
	}

	// --- Layer 4: Watermark video overlay ---
	var overlayConfig VideoOverlayConfig
	if cc.PreviewVideoOverlayConfigJSON != "" && cc.PreviewVideoOverlayConfigJSON != "{}" {
		if err := json.Unmarshal([]byte(cc.PreviewVideoOverlayConfigJSON), &overlayConfig); err != nil {
			log.Warn("effect layer skipped: video overlay (invalid JSON)", "err", err)
		}
	}
	if overlayConfig.Enabled && overlayConfig.Path != "" {
		if _, err := os.Stat(overlayConfig.Path); err == nil {
			extraInputs = append(extraInputs, "-i", overlayConfig.Path)
			overlayOps = append(overlayOps, overlayOp{
				kind:      opWatermark,
				inputIdx:  inputIdx,
				chromaKey: overlayConfig.ChromaKey,
				scalePct:  overlayConfig.ScalePct,
				position:  overlayConfig.Position,
			})
			inputIdx++
			log.Info("effect layer: watermark overlay", "path", overlayConfig.Path)
		} else {
			log.Warn("effect layer skipped: watermark (file not found)", "path", overlayConfig.Path)
		}
	}

	// --- Layer 5: Captions ASS burn-in ---
	if cc.PreviewCaptionsEnabled {
		var captionStyle CaptionTextStyle
		if cc.PreviewCaptionTextStyleJSON != "" && cc.PreviewCaptionTextStyleJSON != "{}" {
			if err := json.Unmarshal([]byte(cc.PreviewCaptionTextStyleJSON), &captionStyle); err != nil {
				log.Warn("caption style JSON parse failed, using defaults", "err", err)
			}
		}
		assPath, cleanup, err := GenerateASS(captionStyle)
		if err != nil {
			log.Warn("effect layer skipped: captions (ASS generation failed)", "err", err)
		} else {
			cleanups = append(cleanups, cleanup)
			escapedPath := strings.ReplaceAll(assPath, "\\", "\\\\")
			escapedPath = strings.ReplaceAll(escapedPath, ":", "\\:")
			simpleFilters = append(simpleFilters, fmt.Sprintf("subtitles=%s", escapedPath))
			log.Info("effect layer: captions", "ass_path", assPath)
		}
	}

	// --- Assemble ffmpeg command ---
	args := assembleFFmpegArgs(req, simpleFilters, extraInputs, overlayOps)
	return args, cleanups
}

// overlayOpKind distinguishes overlay types.
type overlayOpKind int

const (
	opLogo      overlayOpKind = iota
	opWatermark
)

// overlayOp describes a structured overlay operation (logo or watermark).
type overlayOp struct {
	kind      overlayOpKind
	inputIdx  int
	opacity   float64 // logo only
	scale     float64 // logo only
	chromaKey string  // watermark only
	scalePct  int     // watermark only
	position  string
}

// assembleFFmpegArgs builds the final ffmpeg argument list from collected filters and overlays.
func assembleFFmpegArgs(req EffectRequest, simpleFilters []string, extraInputs []string, ops []overlayOp) []string {
	hasOverlays := len(ops) > 0

	if !hasOverlays {
		// Simple case: only -vf filters (eq, crop, subtitles)
		args := []string{"-y", "-ss", fmt.Sprintf("%d", req.StartSec), "-i", req.VideoPath, "-t", fmt.Sprintf("%d", req.DurationSec)}
		if len(simpleFilters) > 0 {
			args = append(args, "-vf", strings.Join(simpleFilters, ","))
		}
		args = append(args, encodingFlags(req.OutputPath)...)
		return args
	}

	// Complex case: need -filter_complex for overlay inputs
	args := []string{"-y", "-ss", fmt.Sprintf("%d", req.StartSec), "-i", req.VideoPath}
	args = append(args, extraInputs...)
	args = append(args, "-t", fmt.Sprintf("%d", req.DurationSec))

	filterComplex := buildFilterComplex(simpleFilters, ops)
	args = append(args, "-filter_complex", filterComplex.chain)
	args = append(args, "-map", filterComplex.videoOut, "-map", "0:a?")
	args = append(args, encodingFlags(req.OutputPath)...)
	return args
}

// filterComplexResult holds the assembled filter_complex string and output label.
type filterComplexResult struct {
	chain    string // semicolon-separated filter chain
	videoOut string // final video output label (e.g., "[afterlogo]")
}

// buildFilterComplex constructs the -filter_complex string with proper label chaining.
func buildFilterComplex(simpleFilters []string, ops []overlayOp) filterComplexResult {
	var parts []string
	currentLabel := "[0:v]"

	// Apply simple filters first (color grading, crop, subtitles)
	if len(simpleFilters) > 0 {
		parts = append(parts, fmt.Sprintf("%s%s[base]", currentLabel, strings.Join(simpleFilters, ",")))
		currentLabel = "[base]"
	}

	// Apply each overlay operation in order
	for _, op := range ops {
		switch op.kind {
		case opLogo:
			outLabel := "[afterlogo]"
			pos := overlayPosition(op.position, op.scale)
			parts = append(parts,
				fmt.Sprintf("[%d]format=rgba,colorchannelmixer=aa=%.2f,scale=iw*%.2f:-1[logo_scaled]",
					op.inputIdx, op.opacity, op.scale),
				fmt.Sprintf("%s[logo_scaled]overlay=%s%s",
					currentLabel, pos, outLabel),
			)
			currentLabel = outLabel

		case opWatermark:
			outLabel := "[afterwatermark]"
			scaleFilter := fmt.Sprintf("scale=iw*%d/100:-1", op.scalePct)
			pos := overlayPosition(op.position, float64(op.scalePct)/100.0)

			if params, ok := chromaKeyParams[strings.ToLower(op.chromaKey)]; ok && strings.ToLower(op.chromaKey) != "none" {
				parts = append(parts,
					fmt.Sprintf("[%d]%s,colorkey=%s:%s:%s,format=rgba[wm_keyed]",
						op.inputIdx, scaleFilter, params[0], params[1], params[2]),
					fmt.Sprintf("%s[wm_keyed]overlay=%s:eof_action=pass%s",
						currentLabel, pos, outLabel),
				)
			} else {
				parts = append(parts,
					fmt.Sprintf("[%d]%s,format=rgba[wm_scaled]",
						op.inputIdx, scaleFilter),
					fmt.Sprintf("%s[wm_scaled]overlay=%s:eof_action=pass%s",
						currentLabel, pos, outLabel),
				)
			}
			currentLabel = outLabel
		}
	}

	return filterComplexResult{
		chain:    strings.Join(parts, ";"),
		videoOut: currentLabel,
	}
}

// overlayPosition maps a named position to ffmpeg overlay coordinates.
func overlayPosition(position string, scale float64) string {
	pos := strings.ToLower(strings.ReplaceAll(position, "_", ""))
	switch pos {
	case "topleft":
		return "10:10"
	case "topcenter":
		return "(W-w)/2:10"
	case "topright":
		return "W-w-10:10"
	case "midleft", "middleleft":
		return "10:(H-h)/2"
	case "midcenter", "center", "middlecenter":
		return "(W-w)/2:(H-h)/2"
	case "midright", "middleright":
		return "W-w-10:(H-h)/2"
	case "bottomleft":
		return "10:H-h-10"
	case "bottomcenter":
		return "(W-w)/2:H-h-10"
	case "bottomright":
		return "W-w-10:H-h-10"
	default:
		return "W-w-10:H-h-10" // default bottom-right
	}
}

// encodingFlags returns the standard output encoding flags.
func encodingFlags(outPath string) []string {
	return []string{
		"-c:v", "libx264", "-preset", "fast", "-crf", "23",
		"-c:a", "aac", "-b:a", "128k",
		"-movflags", "+faststart",
		outPath,
	}
}

// ApplyModeOverrides merges ModeConfig settings from the frontend into a ChannelConfig.
// This allows per-run mode settings (configured on the Mode screen) to override
// the channel's default configuration for preview and clip generation.
func ApplyModeOverrides(cc *database.ChannelConfig, modeCfg json.RawMessage) {
	if len(modeCfg) == 0 || string(modeCfg) == "{}" {
		return
	}

	var mc struct {
		AntiDuplicate *struct {
			Enabled bool   `json:"enabled"`
			Mode    string `json:"mode"`
			Effects *struct {
				SpeedBoost    *bool    `json:"speed_boost"`
				Crop          *bool    `json:"crop"`
				ColorGrading  *bool    `json:"color_grading"`
				Noise         *bool    `json:"noise"`
				NoiseStrength *float64 `json:"noise_strength"`
				Blur          *bool    `json:"blur"`
				BlurEdgePct   *float64 `json:"blur_edge_pct"`
				Zoom          *bool    `json:"zoom"`
			} `json:"effects"`
		} `json:"anti_duplicate"`
		Captions *struct {
			Enabled bool   `json:"enabled"`
			Preset  string `json:"preset"`
		} `json:"captions"`
		Branding *struct {
			LogoEnabled  bool    `json:"logo_enabled"`
			LogoPath     *string `json:"logo_path"`
			LogoPosition string  `json:"logo_position"`
			LogoOpacity  float64 `json:"logo_opacity"`
			LogoScale    float64 `json:"logo_scale"`
		} `json:"branding"`
		VideoOverlay *struct {
			Enabled   bool    `json:"enabled"`
			VideoPath *string `json:"video_path"`
			ScalePct  int     `json:"scale_pct"`
			Position  string  `json:"position"`
			ChromaKey *string `json:"chroma_key"`
		} `json:"video_overlay"`
	}

	if err := json.Unmarshal(modeCfg, &mc); err != nil {
		return // silently skip — invalid ModeConfig won't break the pipeline
	}

	// Anti-duplicate → ChannelConfig visual fields
	if ad := mc.AntiDuplicate; ad != nil && ad.Enabled {
		cc.AntiDuplicateEnabled = true
		cc.AntiDuplicateMode = ad.Mode

		if fx := ad.Effects; fx != nil {
			if fx.ColorGrading != nil && *fx.ColorGrading {
				cc.VisualColorGradingEnabled = true
				// Use subtle defaults if not already set
				if cc.VisualBrightness == 0 {
					cc.VisualBrightness = 0.02
				}
				if cc.VisualSaturation == 0 {
					cc.VisualSaturation = 1.05
				}
				if cc.VisualContrast == 0 {
					cc.VisualContrast = 1.02
				}
			}
			if fx.Crop != nil && *fx.Crop {
				cc.VisualCropEnabled = true
				if cc.VisualCropPercent == 0 {
					cc.VisualCropPercent = 0.02
				}
			}
			if fx.Zoom != nil && *fx.Zoom {
				cc.VisualZoomEnabled = true
				if cc.VisualZoomAmount == 0 {
					cc.VisualZoomAmount = 1.03
				}
			}
			if fx.SpeedBoost != nil && *fx.SpeedBoost {
				cc.SpeedEnabled = true
				if cc.SpeedFactor == 0 {
					cc.SpeedFactor = 1.02
				}
			}
		}
	}

	// Captions
	if cap := mc.Captions; cap != nil {
		cc.PreviewCaptionsEnabled = cap.Enabled
	}

	// Branding
	if br := mc.Branding; br != nil {
		cc.BrandingLogoEnabled = br.LogoEnabled
		if br.LogoPath != nil && *br.LogoPath != "" {
			cc.BrandingLogoPath = sql.NullString{String: *br.LogoPath, Valid: true}
		}
		if br.LogoPosition != "" {
			cc.BrandingLogoPosition = br.LogoPosition
		}
		if br.LogoOpacity > 0 {
			cc.BrandingLogoOpacity = br.LogoOpacity / 100 // frontend sends 0-100, backend uses 0-1
		}
		if br.LogoScale > 0 {
			cc.BrandingLogoScale = br.LogoScale / 100 // frontend sends 5-30 (pct), backend uses 0-1
		}
	}

	// Video overlay
	if vo := mc.VideoOverlay; vo != nil && vo.Enabled {
		overlay := VideoOverlayConfig{
			Enabled:  true,
			ScalePct: vo.ScalePct,
			Position: vo.Position,
		}
		if vo.VideoPath != nil {
			overlay.Path = *vo.VideoPath
		}
		if vo.ChromaKey != nil {
			overlay.ChromaKey = *vo.ChromaKey
		}
		if cfgJSON, err := json.Marshal(overlay); err == nil {
			cc.PreviewVideoOverlayConfigJSON = string(cfgJSON)
		}
	}
}
