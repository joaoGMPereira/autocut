package thumbnail

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Output dimensions for each strategy.
const (
	ShortsWidth  = 1080
	ShortsHeight = 1920

	LandscapeWidth  = 1280
	LandscapeHeight = 720
)

// BuildShortsFilterChain constructs the FFmpeg filter_complex string for the
// shorts thumbnail strategy: blur background + darken + scaled center frame +
// rounded corners + text overlay. Matches the Kotlin FFmpegCenteredThumbnailCreator.
func BuildShortsFilterChain(cfg ThumbnailConfig, fontPath, text string) string {
	centerW := int(float64(ShortsWidth) * cfg.CenterScale)
	centerH := int(float64(centerW) * 9.0 / 16.0) // maintain 16:9 aspect

	// Background: scale → crop → blur → darken
	bg := fmt.Sprintf("[0:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d",
		ShortsWidth, ShortsHeight, ShortsWidth, ShortsHeight)
	if cfg.BlurEnabled {
		bg += fmt.Sprintf(",boxblur=%d", cfg.BlurAmount)
	}
	if cfg.DarkenEnabled {
		bg += fmt.Sprintf(",eq=brightness=-%.2f", cfg.DarkenAmount)
	}
	bg += "[bg]"

	// Center image: scale to configured size
	center := fmt.Sprintf("[0:v]scale=%d:%d:force_original_aspect_ratio=decrease[center_raw]",
		centerW, centerH)

	// Rounded corners via geq alpha mask
	var rounded string
	if cfg.CornerRadius > 0 {
		r := cfg.CornerRadius
		rounded = fmt.Sprintf(
			"[center_raw]format=yuva420p,geq=lum='lum(X,Y)':cb='cb(X,Y)':cr='cr(X,Y)':"+
				"a='if(gt(abs(X-W/2),W/2-%d)*gt(abs(Y-H/2),H/2-%d),"+
				"if(lte(hypot(abs(X-W/2)-(W/2-%d),abs(Y-H/2)-(H/2-%d)),%d),255,0),255)'[rounded]",
			r, r, r, r, r)
	} else {
		rounded = "[center_raw]copy[rounded]"
	}

	// Overlay center on background
	overlay := "[bg][rounded]overlay=(W-w)/2:(H-h)/2"

	// Text overlay
	if text != "" {
		fontSize := cfg.FontSize
		if fontSize == 0 {
			fontSize = autoFontSize(text)
		}

		escapedText := escapeFFmpegText(text)
		escapedFont := strings.ReplaceAll(fontPath, ":", "\\:")
		escapedFont = strings.ReplaceAll(escapedFont, "'", "\\'")

		textColor := strings.TrimPrefix(cfg.TextColor, "#")
		outlineColor := strings.TrimPrefix(cfg.OutlineColor, "#")

		overlay += fmt.Sprintf(
			",drawtext=fontfile='%s':text='%s':fontsize=%d:fontcolor=0x%s:"+
				"x=(w-text_w)/2:y=(h-text_h)/2+%d:borderw=%d:bordercolor=0x%s",
			escapedFont, escapedText, fontSize, textColor,
			cfg.TextYOffset, cfg.StrokeWidth, outlineColor)
	}

	return strings.Join([]string{bg, center, rounded, overlay}, ";\n")
}

// BuildShortsArgs returns the full FFmpeg args for generating a shorts thumbnail
// from a video file at a specific seek time.
func BuildShortsArgs(inputPath string, seekSec float64, outputPath string, filterComplex string) []string {
	return []string{
		"-y",
		"-ss", fmt.Sprintf("%.2f", seekSec),
		"-i", inputPath,
		"-frames:v", "1",
		"-filter_complex", filterComplex,
		"-q:v", "2", // JPEG quality (2 = high)
		outputPath,
	}
}

// autoFontSize calculates font size based on text length.
// Short text gets large font, long text gets smaller.
func autoFontSize(text string) int {
	n := len(text)
	switch {
	case n <= 10:
		return 90
	case n <= 20:
		return 75
	case n <= 40:
		return 60
	case n <= 80:
		return 48
	default:
		return 36
	}
}

// escapeFFmpegText escapes special characters for FFmpeg drawtext filter.
func escapeFFmpegText(s string) string {
	r := strings.NewReplacer(
		"'", "\\'",
		":", "\\:",
		"\\", "\\\\",
		"%", "%%",
	)
	return r.Replace(s)
}

// ShortsStrategy generates 9:16 thumbnails with blur bg + centered frame + text overlay.
type ShortsStrategy struct{}

func (s *ShortsStrategy) Name() string       { return "shorts" }
func (s *ShortsStrategy) Type() StrategyType { return StrategyShorts }

func (s *ShortsStrategy) Generate(ctx context.Context, input StrategyInput) (string, error) {
	if input.SourcePath == "" {
		return "", fmt.Errorf("source path is required")
	}
	if _, err := os.Stat(input.SourcePath); err != nil {
		return "", fmt.Errorf("source not found: %s", input.SourcePath)
	}
	filterComplex := BuildShortsFilterChain(input.Config, input.FontPath, input.Text)
	args := BuildShortsArgs(input.SourcePath, input.SeekSec, input.OutputPath, filterComplex)
	if err := RunFFmpeg(ctx, args, 0, nil); err != nil {
		return "", fmt.Errorf("ffmpeg thumbnail: %w", err)
	}
	return input.OutputPath, nil
}
