package processor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// EffectRequest holds the common config for any video processing job (preview or clip).
type EffectRequest struct {
	VideoPath      string
	StartSec       int
	DurationSec    int64
	ChannelCfg     database.ChannelConfig
	OutputPath     string
	BlurEdgePct    float64 // 0=disabled, 1-100=edge blur percentage
	NoiseStrength  float64 // 0=disabled, 1-10=noise strength
	TranscriptPath string  // optional: path to transcript JSON for real captions
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

// ParseAntiDupExtras extracts blur and noise parameters from a ModeConfig JSON blob.
func ParseAntiDupExtras(modeCfg json.RawMessage) (blurEdgePct, noiseStrength float64) {
	var mc struct {
		AntiDuplicate *struct {
			Enabled bool `json:"enabled"`
			Effects *struct {
				Blur          *bool    `json:"blur"`
				BlurEdgePct   *float64 `json:"blur_edge_pct"`
				Noise         *bool    `json:"noise"`
				NoiseStrength *float64 `json:"noise_strength"`
			} `json:"effects"`
		} `json:"anti_duplicate"`
	}
	if err := json.Unmarshal(modeCfg, &mc); err != nil {
		return
	}
	ad := mc.AntiDuplicate
	if ad == nil || !ad.Enabled {
		return
	}
	fx := ad.Effects
	if fx == nil {
		return
	}
	if fx.Blur != nil && *fx.Blur && fx.BlurEdgePct != nil {
		blurEdgePct = *fx.BlurEdgePct
	}
	if fx.Noise != nil && *fx.Noise && fx.NoiseStrength != nil {
		noiseStrength = *fx.NoiseStrength
	}
	return
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

	// --- Layer 2b: Zoom ---
	if cc.VisualZoomEnabled && cc.VisualZoomAmount > 1.0 {
		z := cc.VisualZoomAmount
		// Scale up then crop back; round to even dimensions to avoid codec errors
		zoom := fmt.Sprintf("scale=iw*%.4f:ih*%.4f,crop=trunc(iw/%.4f/2)*2:trunc(ih/%.4f/2)*2", z, z, z, z)
		simpleFilters = append(simpleFilters, zoom)
		log.Info("effect layer: zoom", "amount", z)
	}

	// --- Layer 2c: Speed ---
	if cc.SpeedEnabled && cc.SpeedFactor > 0 && cc.SpeedFactor != 1.0 {
		simpleFilters = append(simpleFilters, fmt.Sprintf("setpts=PTS/%.4f", cc.SpeedFactor))
		log.Info("effect layer: speed", "factor", cc.SpeedFactor)
	}

	// --- Layer 2d: Blur background ---
	// Prepend to overlayOps so it runs as a filter_complex split BEFORE logo and video overlay.
	// The effect: raw video blurred + scaled-to-fill as background; processed video
	// scaled down as foreground, overlaid centered. This creates the "portrait blur" look.
	if req.BlurEdgePct > 0 {
		bgSpeedFactor := 0.0
		if cc.SpeedEnabled && cc.SpeedFactor > 0 && cc.SpeedFactor != 1.0 {
			bgSpeedFactor = cc.SpeedFactor
		}
		overlayOps = append([]overlayOp{{kind: opBlurBg, blurEdgePct: req.BlurEdgePct, speedFactor: bgSpeedFactor}}, overlayOps...)
		log.Info("effect layer: blur background", "edge_pct", req.BlurEdgePct, "speed_factor", bgSpeedFactor)
	}

	// --- Layer 2e: Noise ---
	if req.NoiseStrength > 0 {
		n := int(3 + (req.NoiseStrength/10.0)*27)
		simpleFilters = append(simpleFilters, fmt.Sprintf("noise=alls=%d:allf=t", n))
		log.Info("effect layer: noise", "strength", req.NoiseStrength, "ffmpeg_n", n)
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
	} else {
		log.Debug("effect layer skipped: video overlay (no config)", "json_len", len(cc.PreviewVideoOverlayConfigJSON))
	}
	if overlayConfig.Enabled && overlayConfig.Path != "" {
		if _, err := os.Stat(overlayConfig.Path); err == nil {
			extraInputs = append(extraInputs, "-i", overlayConfig.Path)
			overlayOps = append(overlayOps, overlayOp{
				kind:      opVideoOverlay,
				inputIdx:  inputIdx,
				chromaKey: overlayConfig.ChromaKey,
				scalePct:  overlayConfig.ScalePct,
				position:  overlayConfig.Position,
			})
			inputIdx++
			log.Info("effect layer: video overlay", "path", overlayConfig.Path)
		} else {
			log.Warn("effect layer skipped: video overlay (file not found)", "path", overlayConfig.Path)
		}
	}

	// postFilters are applied AFTER all overlay compositing (blur-bg, logo, video overlay).
	// This ensures subtitles and drawtext render at full resolution on the final composited frame,
	// not on the scaled-down foreground stream (which would make them tiny under blur-bg).
	var postFilters []string

	// --- Layer 5: Captions ASS burn-in ---
	if cc.PreviewCaptionsEnabled {
		var captionStyle CaptionTextStyle
		if cc.PreviewCaptionTextStyleJSON != "" && cc.PreviewCaptionTextStyleJSON != "{}" {
			if err := json.Unmarshal([]byte(cc.PreviewCaptionTextStyleJSON), &captionStyle); err != nil {
				log.Warn("caption style JSON parse failed, using defaults", "err", err)
			}
		}
		captionSpeed := 1.0
		if cc.SpeedEnabled && cc.SpeedFactor > 0 {
			captionSpeed = cc.SpeedFactor
		}
		assPath, cleanup, err := GenerateASSFromTranscript(req.TranscriptPath, req.StartSec, req.DurationSec, captionStyle, captionSpeed)
		if err != nil {
			log.Warn("effect layer skipped: captions (ASS generation failed)", "err", err)
		} else {
			cleanups = append(cleanups, cleanup)
			escapedPath := strings.ReplaceAll(assPath, "\\", "\\\\")
			escapedPath = strings.ReplaceAll(escapedPath, ":", "\\:")
			postFilters = append(postFilters, fmt.Sprintf("subtitles=filename=%s", escapedPath))
			log.Info("effect layer: captions", "ass_path", assPath)
		}
	}

	// --- Layer 6: Text overlays ---
	if cc.PreviewTextOverlayConfigJSON != "" && cc.PreviewTextOverlayConfigJSON != "[]" {
		var items []textOverlayItem
		if err := json.Unmarshal([]byte(cc.PreviewTextOverlayConfigJSON), &items); err == nil {
			for _, item := range items {
				filter := buildDrawtextFilter(item, req.DurationSec)
				if filter != "" {
					postFilters = append(postFilters, filter)
					log.Info("effect layer: text overlay", "text", item.Text)
				}
			}
		}
	}

	// --- Assemble ffmpeg command ---
	args := assembleFFmpegArgs(req, simpleFilters, postFilters, extraInputs, overlayOps)
	return args, cleanups
}

// overlayOpKind distinguishes overlay types.
type overlayOpKind int

const (
	opBlurBg       overlayOpKind = iota // blur-background: smaller video over blurred bg
	opLogo                              // static image watermark (channel avatar/branding logo)
	opVideoOverlay                      // video overlay (e.g. subscribe animation)
)

// overlayOp describes a structured overlay operation (blur-bg, logo, or video overlay).
type overlayOp struct {
	kind        overlayOpKind
	inputIdx    int
	opacity     float64 // logo only
	scale       float64 // logo only
	chromaKey   string  // video overlay only
	scalePct    int     // video overlay only
	position    string
	blurEdgePct float64 // blurBg only
	speedFactor float64 // blurBg only: match bg speed to fg speed
}

// assembleFFmpegArgs builds the final ffmpeg argument list from collected filters and overlays.
// postFilters (subtitles, drawtext) are applied after all overlay compositing so they render
// at full resolution on the final composited frame.
func assembleFFmpegArgs(req EffectRequest, simpleFilters []string, postFilters []string, extraInputs []string, ops []overlayOp) []string {
	hasOverlays := len(ops) > 0

	if !hasOverlays {
		// Simple case: no overlays — all filters go directly into -vf
		allFilters := append(simpleFilters, postFilters...)
		args := []string{"-y", "-ss", fmt.Sprintf("%d", req.StartSec), "-i", req.VideoPath, "-t", fmt.Sprintf("%d", req.DurationSec)}
		if len(allFilters) > 0 {
			args = append(args, "-vf", strings.Join(allFilters, ","))
		}
		if req.ChannelCfg.SpeedEnabled && req.ChannelCfg.SpeedFactor > 0 && req.ChannelCfg.SpeedFactor != 1.0 {
			args = append(args, "-af", fmt.Sprintf("atempo=%.4f", req.ChannelCfg.SpeedFactor))
		}
		args = append(args, encodingFlags(req.OutputPath)...)
		return args
	}

	// Complex case: need -filter_complex for overlay inputs
	args := []string{"-y", "-ss", fmt.Sprintf("%d", req.StartSec), "-i", req.VideoPath}
	args = append(args, extraInputs...)
	args = append(args, "-t", fmt.Sprintf("%d", req.DurationSec))

	filterComplex := buildFilterComplex(simpleFilters, postFilters, ops)
	args = append(args, "-filter_complex", filterComplex.chain)
	args = append(args, "-map", filterComplex.videoOut, "-map", "0:a?")
	if req.ChannelCfg.SpeedEnabled && req.ChannelCfg.SpeedFactor > 0 && req.ChannelCfg.SpeedFactor != 1.0 {
		args = append(args, "-af", fmt.Sprintf("atempo=%.4f", req.ChannelCfg.SpeedFactor))
	}
	args = append(args, encodingFlags(req.OutputPath)...)
	return args
}

// filterComplexResult holds the assembled filter_complex string and output label.
type filterComplexResult struct {
	chain    string // semicolon-separated filter chain
	videoOut string // final video output label (e.g., "[afterlogo]")
}

// buildFilterComplex constructs the -filter_complex string with proper label chaining.
// postFilters (subtitles, drawtext) are applied last, after all overlay compositing.
func buildFilterComplex(simpleFilters []string, postFilters []string, ops []overlayOp) filterComplexResult {
	var parts []string

	// If the first op is blur-bg, pre-split [0:v] so the bg stream and the fg stream
	// (which gets simple filters applied) are independent — avoids double-consumption
	// of [0:v] in the same filter_complex graph.
	hasBlurBg := len(ops) > 0 && ops[0].kind == opBlurBg
	var bgRawStream string
	currentLabel := "[0:v]"

	if hasBlurBg {
		bgRawStream = "[0_bg_raw]"
		fgRawStream := "[0_fg_raw]"
		parts = append(parts, fmt.Sprintf("[0:v]split%s%s", bgRawStream, fgRawStream))
		currentLabel = fgRawStream
	}

	// Apply simple filters (color grading, crop, captions, text overlays…)
	if len(simpleFilters) > 0 {
		parts = append(parts, fmt.Sprintf("%s%s[base]", currentLabel, strings.Join(simpleFilters, ",")))
		currentLabel = "[base]"
	}

	// Apply each overlay operation in order
	for i, op := range ops {
		switch op.kind {
		case opBlurBg:
			// Blur-background: raw stream → blur + scale-to-fill as BG;
			// processed stream (currentLabel) → scale down as FG; overlay centered.
			bgBlurred := fmt.Sprintf("[_bbg%d]", i)
			fgSmall := fmt.Sprintf("[_bfg%d]", i)
			outLabel := fmt.Sprintf("[blur_out%d]", i)

			e := op.blurEdgePct / 100.0
			fgScale := 1.0 - 2*e
			if fgScale < 0.5 {
				fgScale = 0.5
			}
			sigma := 20.0 + e*30.0 // sigma 20–50: heavier blur for wider edges

			bgFilter := fmt.Sprintf("gblur=sigma=%.0f,scale=iw:ih:force_original_aspect_ratio=increase,crop=iw:ih", sigma)
			if op.speedFactor > 0 && op.speedFactor != 1.0 {
				bgFilter = fmt.Sprintf("setpts=PTS/%.4f,%s", op.speedFactor, bgFilter)
			}
			parts = append(parts,
				// BG: blur the raw (unprocessed) stream and scale to fill the frame
				fmt.Sprintf("%s%s%s", bgRawStream, bgFilter, bgBlurred),
				// FG: scale down the processed stream (has color grading, captions, etc.)
				fmt.Sprintf("%sscale=trunc(iw*%.4f/2)*2:trunc(ih*%.4f/2)*2%s",
					currentLabel, fgScale, fgScale, fgSmall),
				// Overlay FG centered on blurred BG
				fmt.Sprintf("%s%soverlay=(W-w)/2:(H-h)/2%s", bgBlurred, fgSmall, outLabel),
			)
			currentLabel = outLabel

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

		case opVideoOverlay:
			outLabel := "[afterovl]"
			scaleFilter := fmt.Sprintf("scale=iw*%d/100:-1", op.scalePct)
			pos := overlayPosition(op.position, float64(op.scalePct)/100.0)

			if params, ok := chromaKeyParams[strings.ToLower(op.chromaKey)]; ok && strings.ToLower(op.chromaKey) != "none" {
				parts = append(parts,
					fmt.Sprintf("[%d]%s,colorkey=%s:%s:%s,format=rgba[ovl_keyed]",
						op.inputIdx, scaleFilter, params[0], params[1], params[2]),
					fmt.Sprintf("%s[ovl_keyed]overlay=%s:eof_action=pass%s",
						currentLabel, pos, outLabel),
				)
			} else {
				parts = append(parts,
					fmt.Sprintf("[%d]%s,format=rgba[ovl_scaled]",
						op.inputIdx, scaleFilter),
					fmt.Sprintf("%s[ovl_scaled]overlay=%s:eof_action=pass%s",
						currentLabel, pos, outLabel),
				)
			}
			currentLabel = outLabel
		}
	}

	// Apply post-filters (subtitles, drawtext) to the fully composited output.
	// These must come AFTER blur-bg/logo/overlay so they render at full frame resolution.
	if len(postFilters) > 0 {
		outLabel := "[final_post]"
		parts = append(parts, fmt.Sprintf("%s%s%s", currentLabel, strings.Join(postFilters, ","), outLabel))
		currentLabel = outLabel
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
			Enabled        bool    `json:"enabled"`
			Preset         string  `json:"preset"`
			FontFamily     *string `json:"font_family"`
			Bold           *bool   `json:"bold"`
			Italic         *bool   `json:"italic"`
			Uppercase      *bool   `json:"uppercase"`
			FontSize       *int    `json:"font_size"`
			TextColor      *string `json:"text_color"`
			BgEnabled      *bool   `json:"bg_enabled"`
			OutlineEnabled *bool   `json:"outline_enabled"`
			OutlineColor   *string `json:"outline_color"`
			OutlineWidth   *int    `json:"outline_width"`
			ShadowEnabled  *bool   `json:"shadow_enabled"`
			ShadowColor    *string `json:"shadow_color"`
			ShadowDistance *int    `json:"shadow_distance"`
			VerticalOffset *int    `json:"vertical_offset"`
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
		TextOverlays *struct {
			Enabled bool            `json:"enabled"`
			Items   json.RawMessage `json:"items"`
		} `json:"text_overlays"`
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
		if cap.Enabled {
			style := CaptionTextStyle{Alignment: 2}
			if cap.FontFamily != nil {
				style.FontFamily = *cap.FontFamily
			}
			if cap.FontSize != nil {
				style.FontSize = *cap.FontSize
			}
			if cap.TextColor != nil {
				style.PrimaryColor = strings.TrimPrefix(*cap.TextColor, "#")
			}
			if cap.Bold != nil {
				style.Bold = *cap.Bold
			}
			if cap.Italic != nil {
				style.Italic = *cap.Italic
			}
			if cap.OutlineEnabled != nil && *cap.OutlineEnabled {
				if cap.OutlineColor != nil {
					style.OutlineColor = strings.TrimPrefix(*cap.OutlineColor, "#")
				}
				if cap.OutlineWidth != nil {
					style.OutlineWidth = *cap.OutlineWidth
				}
			}
			if cap.BgEnabled != nil {
				style.BgEnabled = *cap.BgEnabled
			}
			if cap.ShadowEnabled != nil && *cap.ShadowEnabled {
				if cap.ShadowDistance != nil {
					style.Shadow = *cap.ShadowDistance
				}
			}
			if cap.VerticalOffset != nil {
				style.MarginV = *cap.VerticalOffset
			}
			if styleJSON, err := json.Marshal(style); err == nil {
				cc.PreviewCaptionTextStyleJSON = string(styleJSON)
			}
		}
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

	// Text overlays
	if to := mc.TextOverlays; to != nil && to.Enabled && len(to.Items) > 0 {
		cc.PreviewTextOverlayConfigJSON = string(to.Items)
	}
}

// --- Text overlay types and helpers ---

type textOverlayStyle struct {
	FontSize        int     `json:"fontSize"`
	FontFamily      string  `json:"fontFamily"`
	TextColor       string  `json:"textColor"`
	IsBold          bool    `json:"isBold"`
	IsItalic        bool    `json:"isItalic"`
	AllCaps         bool    `json:"allCaps"`
	BackgroundColor *string `json:"backgroundColor"`
	ShadowColor     *string `json:"shadowColor"`
	ShadowOffset    int     `json:"shadowOffset"`
}

type textOverlayItem struct {
	Text      string            `json:"text"`
	ApplyFull bool              `json:"apply_full"`
	StartSec  float64           `json:"start_sec"`
	EndSec    float64           `json:"end_sec"`
	Position  string            `json:"position"`
	Style     *textOverlayStyle `json:"style"`
}

func buildDrawtextFilter(item textOverlayItem, _ int64) string {
	text := item.Text
	if text == "" {
		return ""
	}

	// Style defaults
	fontSize := 48
	fontColor := "white"
	fontFamily := "Arial"
	if s := item.Style; s != nil {
		if s.FontSize > 0 {
			fontSize = s.FontSize
		}
		if s.TextColor != "" {
			fontColor = s.TextColor
		}
		if s.FontFamily != "" {
			fontFamily = s.FontFamily
		}
		if s.AllCaps {
			text = strings.ToUpper(text)
		}
	}

	// Escape special chars for drawtext
	text = strings.ReplaceAll(text, "'", "'\\''")
	text = strings.ReplaceAll(text, ":", "\\:")

	// Position -> x:y expressions
	x, y := drawtextPosition(item.Position)

	// Timing
	enable := ""
	if !item.ApplyFull {
		enable = fmt.Sprintf(":enable='between(t,%.1f,%.1f)'", item.StartSec, item.EndSec)
	}

	// Append Bold/Italic suffix to font name — drawtext has no fontstyle option
	if s := item.Style; s != nil {
		if s.IsBold && s.IsItalic {
			fontFamily += " Bold Italic"
		} else if s.IsBold {
			fontFamily += " Bold"
		} else if s.IsItalic {
			fontFamily += " Italic"
		}
	}

	filter := fmt.Sprintf("drawtext=text='%s':fontsize=%d:fontcolor=%s:font='%s':x=%s:y=%s%s",
		text, fontSize, fontColor, fontFamily, x, y, enable)

	// Shadow
	if s := item.Style; s != nil && s.ShadowColor != nil && *s.ShadowColor != "" {
		filter += fmt.Sprintf(":shadowcolor=%s:shadowx=%d:shadowy=%d", *s.ShadowColor, s.ShadowOffset, s.ShadowOffset)
	}

	// Background box
	if s := item.Style; s != nil && s.BackgroundColor != nil && *s.BackgroundColor != "" {
		filter += fmt.Sprintf(":box=1:boxcolor=%s@0.85:boxborderw=12", *s.BackgroundColor)
	}

	return filter
}

func drawtextPosition(position string) (x, y string) {
	pos := strings.ToLower(strings.ReplaceAll(position, "_", ""))
	switch pos {
	case "topleft":
		return "20", "20"
	case "topcenter":
		return "(w-tw)/2", "20"
	case "topright":
		return "w-tw-20", "20"
	case "midleft", "middleleft":
		return "20", "(h-th)/2"
	case "midcenter", "center", "middlecenter":
		return "(w-tw)/2", "(h-th)/2"
	case "midright", "middleright":
		return "w-tw-20", "(h-th)/2"
	case "bottomleft":
		return "20", "h-th-20"
	case "bottomcenter":
		return "(w-tw)/2", "h-th-20"
	case "bottomright":
		return "w-tw-20", "h-th-20"
	default:
		return "(w-tw)/2", "(h-th)/2"
	}
}

// EnrichChannelConfig resolves relative asset paths and applies fallbacks
// so that BuildEffectChain receives fully-resolved config.
// Must be called AFTER ApplyModeOverrides.
func EnrichChannelConfig(cc *database.ChannelConfig, overlayLibraryPath, dataDir string, channelID int64) {
	resolveVideoOverlayPath(cc, overlayLibraryPath, dataDir)
	resolveLogoFallback(cc, dataDir, channelID)
}

// resolveVideoOverlayPath patches cc.PreviewVideoOverlayConfigJSON so that relative
// video paths are resolved against dataDir. This handles the case where the frontend
// stores just a filename (e.g. "seguir-1.mp4") without a full path.
// Search order: overlayLibraryPath, then dataDir.
func resolveVideoOverlayPath(cc *database.ChannelConfig, overlayLibraryPath, dataDir string) {
	if cc.PreviewVideoOverlayConfigJSON == "" || cc.PreviewVideoOverlayConfigJSON == "{}" {
		return
	}
	var cfg VideoOverlayConfig
	if err := json.Unmarshal([]byte(cc.PreviewVideoOverlayConfigJSON), &cfg); err != nil {
		return
	}
	if !cfg.Enabled || cfg.Path == "" {
		return
	}
	if filepath.IsAbs(cfg.Path) {
		return
	}
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

// resolveLogoFallback uses the channel avatar as logo when logo is enabled but path is missing.
func resolveLogoFallback(cc *database.ChannelConfig, dataDir string, channelID int64) {
	if !cc.BrandingLogoEnabled || (cc.BrandingLogoPath.Valid && cc.BrandingLogoPath.String != "") || channelID <= 0 {
		return
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png"} {
		p := filepath.Join(dataDir, "avatars", fmt.Sprintf("%d%s", channelID, ext))
		if _, err := os.Stat(p); err == nil {
			cc.BrandingLogoPath = sql.NullString{String: p, Valid: true}
			return
		}
	}
}
