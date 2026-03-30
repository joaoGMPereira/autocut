package thumbnail

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// ThumbnailGenerator wraps FFmpeg (and optionally ImageMagick) to produce
// YouTube thumbnails from video frames and background images.
//
// Kotlin ref: FFmpegThumbnailUtils facade + FFmpegBrandedThumbnailCreator +
//             FFmpegCenteredThumbnailCreator + FFmpegFrameExtractor
type ThumbnailGenerator struct {
	ffmpegPath  string
	magickPath  string
	ffprobePath string
	exec        Executor
	log         *slog.Logger
}

// New returns a ThumbnailGenerator using the supplied binary names.
// Pass empty strings to use the defaults ("ffmpeg" / "convert").
//
// Kotlin ref: FFmpegThumbnailUtils object initialisation
func New(ffmpegPath, magickPath string) *ThumbnailGenerator {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	if magickPath == "" {
		magickPath = "convert"
	}
	return &ThumbnailGenerator{
		ffmpegPath:  ffmpegPath,
		magickPath:  magickPath,
		ffprobePath: "ffprobe",
		exec:        DefaultExecutor{},
		log:         slog.With("component", "thumbnail.generator"),
	}
}

// newWithExec is used by tests to inject a mock Executor.
func newWithExec(ffmpegPath, magickPath string, exec Executor) *ThumbnailGenerator {
	g := New(ffmpegPath, magickPath)
	g.exec = exec
	return g
}

// ExtractBestFrame seeks to 10 % of the video duration and captures one frame.
//
// Kotlin ref: FFmpegFrameExtractor.extractFrame + getVideoDuration
func (g *ThumbnailGenerator) ExtractBestFrame(videoPath, output string) error {
	dur, err := g.videoDuration(videoPath)
	if err != nil {
		g.log.Warn("could not detect duration, defaulting seek to 0s",
			"op", "ExtractBestFrame", "video", videoPath, "err", err)
		dur = 0
	}

	seek := dur * 0.1
	seekStr := strconv.FormatFloat(seek, 'f', 3, 64)

	args := []string{
		"-y",
		"-ss", seekStr,
		"-i", videoPath,
		"-frames:v", "1",
		output,
	}
	if out, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		return fmt.Errorf("ExtractBestFrame: ffmpeg: %w\noutput: %s", err, out)
	}
	return nil
}

// GenerateBranded creates a YouTube-style branded thumbnail.
//
// When cfg.FontPath is non-empty the generator attempts to use ImageMagick
// (convert) for compositing; otherwise it falls back to the pure-FFmpeg
// drawtext approach.
//
// Kotlin ref: FFmpegBrandedThumbnailCreator.createAdvanced +
//             BrandedStrategy.renderHybrid
func (g *ThumbnailGenerator) GenerateBranded(cfg BrandedConfig, output string) error {
	cfg = cfg.withDefaults()

	if cfg.FontPath != "" {
		return g.generateBrandedMagick(cfg, output)
	}
	return g.generateBrandedFFmpeg(cfg, output)
}

// generateBrandedFFmpeg renders a branded thumbnail using only FFmpeg drawtext.
// Kotlin ref: FFmpegBrandedRenderer.render (FFmpeg-only path)
func (g *ThumbnailGenerator) generateBrandedFFmpeg(cfg BrandedConfig, output string) error {
	text := escapeDrawtext(cfg.TextOverlay)
	xExpr := cfg.Position.ffmpegX()
	yExpr := cfg.Position.ffmpegY()

	filter := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,"+
			"drawtext=text='%s':fontcolor=%s:fontsize=%d:x=%s:y=%s",
		cfg.Width, cfg.Height, cfg.Width, cfg.Height,
		text, cfg.FontColor, cfg.FontSize, xExpr, yExpr,
	)

	args := []string{
		"-y",
		"-i", cfg.BackgroundPath,
		"-vf", filter,
		"-q:v", "2",
		output,
	}
	if out, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		return fmt.Errorf("GenerateBranded (ffmpeg): %w\noutput: %s", err, out)
	}
	return nil
}

// generateBrandedMagick renders using ImageMagick composite + text.
// Kotlin ref: ImageMagickBrandedRenderer.render (ImageMagick path)
func (g *ThumbnailGenerator) generateBrandedMagick(cfg BrandedConfig, output string) error {
	// ImageMagick: resize background then draw text with the supplied font.
	sizeStr := fmt.Sprintf("%dx%d!", cfg.Width, cfg.Height)
	args := []string{
		cfg.BackgroundPath,
		"-resize", sizeStr,
		"-font", cfg.FontPath,
		"-pointsize", strconv.Itoa(cfg.FontSize),
		"-fill", cfg.FontColor,
		"-gravity", anchorToMagickGravity(cfg.Position.Anchor),
		"-annotate", fmt.Sprintf("%+d%+d", cfg.Position.X, cfg.Position.Y),
		cfg.TextOverlay,
		output,
	}
	if out, err := g.exec.Run(g.magickPath, args...); err != nil {
		return fmt.Errorf("GenerateBranded (magick): %w\noutput: %s", err, out)
	}
	return nil
}

// GenerateCentered creates a thumbnail with the source frame centred over a
// blurred copy of itself — the classic Shorts layout.
//
// Kotlin ref: FFmpegCenteredThumbnailCreator.create
func (g *ThumbnailGenerator) GenerateCentered(cfg CenteredConfig, output string) error {
	cfg = cfg.withDefaults()

	centerWidth := cfg.Width * 3 / 4 // ~75 % — matches Kotlin default
	centerHeight := centerWidth * 9 / 16

	var filterParts []string

	// Background stream: scale + crop + blur + slight darken
	bgFilter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,boxblur=25,eq=brightness=-0.3[bg]",
		cfg.Width, cfg.Height, cfg.Width, cfg.Height,
	)
	filterParts = append(filterParts, bgFilter)

	// Centre stream: resize to 75 %
	centerFilter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease[center]",
		centerWidth, centerHeight,
	)
	filterParts = append(filterParts, centerFilter)

	// Overlay
	overlay := "[bg][center]overlay=(W-w)/2:(H-h)/2"

	// Optional text
	if cfg.TextOverlay != "" {
		text := escapeDrawtext(cfg.TextOverlay)
		overlay += fmt.Sprintf(
			",drawtext=text='%s':fontcolor=%s:fontsize=%d:x=(w-text_w)/2:y=(h-text_h)/2",
			text, cfg.FontColor, cfg.FontSize,
		)
		if cfg.FontPath != "" {
			overlay += fmt.Sprintf(":fontfile='%s'", cfg.FontPath)
		}
	}

	filterComplex := strings.Join(filterParts, ";") + ";" + overlay

	args := []string{
		"-y",
		"-i", cfg.FramePath,
		"-filter_complex", filterComplex,
		"-q:v", "2",
		output,
	}
	if out, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		return fmt.Errorf("GenerateCentered: %w\noutput: %s", err, out)
	}
	return nil
}

// Preview generates a 320x180 scaled-down preview image.
//
// Kotlin ref: FFmpegFrameExtractor.resizeImage (preview use-case)
func (g *ThumbnailGenerator) Preview(inputPath, output string) error {
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", "scale=320:180",
		"-frames:v", "1",
		output,
	}
	if out, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		return fmt.Errorf("Preview: %w\noutput: %s", err, out)
	}
	return nil
}

// --- internal helpers ---

// videoDuration uses ffprobe to read the duration of a media file.
// Kotlin ref: FFmpegFrameExtractor.getVideoDuration
func (g *ThumbnailGenerator) videoDuration(path string) (float64, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		path,
	}
	out, err := g.exec.Run(g.ffprobePath, args...)
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	var probe struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		return 0, fmt.Errorf("parse ffprobe json: %w", err)
	}

	dur, err := strconv.ParseFloat(strings.TrimSpace(probe.Format.Duration), 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", probe.Format.Duration, err)
	}
	return dur, nil
}

// escapeDrawtext escapes characters that would break FFmpeg's drawtext filter.
// Kotlin ref: FFmpegBrandedThumbnailCreator.createAdvanced (escaping block)
func escapeDrawtext(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ":", `\:`)
	s = strings.ReplaceAll(s, "'", `\'`)
	return s
}

// anchorToMagickGravity maps an AnchorXxx constant to an ImageMagick gravity string.
func anchorToMagickGravity(anchor string) string {
	switch anchor {
	case AnchorTopLeft:
		return "NorthWest"
	case AnchorTopCenter:
		return "North"
	case AnchorTopRight:
		return "NorthEast"
	case AnchorBottomLeft:
		return "SouthWest"
	case AnchorBottomCenter:
		return "South"
	case AnchorBottomRight:
		return "SouthEast"
	default: // AnchorCenter or ""
		return "Center"
	}
}
