package thumbnail

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LongFormConfig holds parameters for a YouTube long-form thumbnail.
// Kotlin ref: LongFormThumbnailStrategy config
type LongFormConfig struct {
	VideoPath   string
	TextOverlay string
	FontPath    string
	FontColor   string
	FontSize    int
	NumFrames   int // default 5
	Width       int // default 1280
	Height      int // default 720
}

// withDefaults returns a copy of c with zero-valued fields filled in.
func (c LongFormConfig) withDefaults() LongFormConfig {
	if c.NumFrames <= 0 {
		c.NumFrames = 5
	}
	if c.Width == 0 {
		c.Width = 1280
	}
	if c.Height == 0 {
		c.Height = 720
	}
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	if c.FontSize == 0 {
		c.FontSize = 80
	}
	return c
}

// ShortsThumbnailConfig holds parameters for a YouTube Shorts thumbnail.
// Kotlin ref: ShortsThumbnailStrategy config
type ShortsThumbnailConfig struct {
	VideoPath   string
	TextOverlay string
	FontPath    string
	FontColor   string
	FontSize    int
	Width       int // default 1080
	Height      int // default 1920
}

// withDefaults returns a copy of c with zero-valued fields filled in.
func (c ShortsThumbnailConfig) withDefaults() ShortsThumbnailConfig {
	if c.Width == 0 {
		c.Width = 1080
	}
	if c.Height == 0 {
		c.Height = 1920
	}
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	if c.FontSize == 0 {
		c.FontSize = 70
	}
	return c
}

// ---------------------------------------------------------------------------
// GenerateLongForm
// ---------------------------------------------------------------------------

// GenerateLongForm generates a YouTube-style long-form thumbnail by:
//  1. Extracting NumFrames frames uniformly distributed through the video
//  2. Selecting the best frame (largest file = more visual information)
//  3. Generating a branded thumbnail using the selected frame
//
// Kotlin ref: LongFormThumbnailStrategy.generate
func (g *ThumbnailGenerator) GenerateLongForm(cfg LongFormConfig, output string) error {
	cfg = cfg.withDefaults()

	dur, err := g.getVideoDuration(cfg.VideoPath)
	if err != nil {
		g.log.Warn("GenerateLongForm: duration detection failed, using 60s fallback",
			"op", "long_form", "video", cfg.VideoPath, "err", err)
		dur = 60 * time.Second
	}

	tmpDir, err := os.MkdirTemp("", "autocut_longform_*")
	if err != nil {
		return fmt.Errorf("generate long form: mktemp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract frames uniformly distributed across the video.
	framePaths := make([]string, cfg.NumFrames)
	for i := 0; i < cfg.NumFrames; i++ {
		// Distribute timestamps: e.g. for 5 frames at 10%, 30%, 50%, 70%, 90%
		fraction := float64(i+1) / float64(cfg.NumFrames+1)
		ts := time.Duration(float64(dur) * fraction)

		framePaths[i] = filepath.Join(tmpDir, fmt.Sprintf("frame_%d.jpg", i))
		if err := g.extractFrame(cfg.VideoPath, ts, framePaths[i]); err != nil {
			g.log.Warn("GenerateLongForm: frame extraction failed",
				"op", "long_form", "frame", i, "ts", ts, "err", err)
			// Continue — we'll pick best from whatever succeeded
		}
	}

	// Select best frame (largest file size = richest visual content).
	bestFrame := selectBestFrame(framePaths)
	if bestFrame == "" {
		return fmt.Errorf("generate long form: no frames extracted from %q", cfg.VideoPath)
	}

	g.log.Debug("GenerateLongForm: best frame selected",
		"op", "long_form", "frame", bestFrame)

	// Render thumbnail using the branded generator.
	brandedCfg := BrandedConfig{
		BackgroundPath: bestFrame,
		TextOverlay:    cfg.TextOverlay,
		FontPath:       cfg.FontPath,
		FontColor:      cfg.FontColor,
		FontSize:       cfg.FontSize,
		Width:          cfg.Width,
		Height:         cfg.Height,
		Position:       Position{Anchor: AnchorBottomCenter},
	}
	return g.GenerateBranded(brandedCfg, output)
}

// ---------------------------------------------------------------------------
// GenerateShortsThumbnail
// ---------------------------------------------------------------------------

// GenerateShortsThumbnail generates a 9:16 thumbnail for YouTube Shorts by:
//  1. Extracting the middle frame of the video
//  2. Applying a 9:16 crop + scale to 1080x1920
//  3. Optionally overlaying text
//
// Kotlin ref: ShortsThumbnailStrategy.generate
func (g *ThumbnailGenerator) GenerateShortsThumbnail(cfg ShortsThumbnailConfig, output string) error {
	cfg = cfg.withDefaults()

	dur, err := g.getVideoDuration(cfg.VideoPath)
	if err != nil {
		g.log.Warn("GenerateShortsThumbnail: duration detection failed, using 0s",
			"op", "shorts_thumbnail", "video", cfg.VideoPath, "err", err)
		dur = 0
	}

	midTS := dur / 2

	tmpDir, err := os.MkdirTemp("", "autocut_shorts_*")
	if err != nil {
		return fmt.Errorf("generate shorts thumbnail: mktemp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	rawFrame := filepath.Join(tmpDir, "raw_frame.jpg")
	if err := g.extractFrame(cfg.VideoPath, midTS, rawFrame); err != nil {
		return fmt.Errorf("generate shorts thumbnail: extract frame: %w", err)
	}

	// Build filter: crop to 9:16 aspect ratio, then scale to target dimensions.
	filter := fmt.Sprintf("crop=ih*9/16:ih,scale=%d:%d", cfg.Width, cfg.Height)

	// Append drawtext if TextOverlay is set.
	if cfg.TextOverlay != "" {
		text := escapeDrawtext(cfg.TextOverlay)
		dtFilter := fmt.Sprintf(
			"drawtext=text='%s':fontcolor=%s:fontsize=%d:x=(w-text_w)/2:y=(h-text_h)*0.85",
			text, cfg.FontColor, cfg.FontSize,
		)
		if cfg.FontPath != "" {
			dtFilter += fmt.Sprintf(":fontfile='%s'", cfg.FontPath)
		}
		filter += "," + dtFilter
	}

	args := []string{
		"-y",
		"-i", rawFrame,
		"-vf", filter,
		"-q:v", "2",
		output,
	}

	g.log.Debug("GenerateShortsThumbnail", "op", "shorts_thumbnail",
		"video", cfg.VideoPath, "output", output)
	if out, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		return fmt.Errorf("GenerateShortsThumbnail: ffmpeg: %w\noutput: %s", err, out)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// getVideoDuration returns the duration of a media file using ffprobe.
// Falls back to ffmpeg -i parsing when ffprobe is unavailable.
// Kotlin ref: FFmpegFrameExtractor.getVideoDuration
func (g *ThumbnailGenerator) getVideoDuration(videoPath string) (time.Duration, error) {
	// videoDuration returns float64 seconds — convert to Duration.
	secs, err := g.videoDuration(videoPath)
	if err != nil {
		return 0, err
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// extractFrame extracts a single frame at the given timestamp.
// Kotlin ref: FFmpegFrameExtractor.extractFrame
func (g *ThumbnailGenerator) extractFrame(videoPath string, timestamp time.Duration, outputPath string) error {
	tsStr := fmt.Sprintf("%.3f", timestamp.Seconds())

	args := []string{
		"-y",
		"-ss", tsStr,
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "2",
		outputPath,
	}
	if out, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		return fmt.Errorf("extractFrame: ffmpeg: %w\noutput: %s", err, out)
	}
	return nil
}

// selectBestFrame returns the path of the frame file with the largest size.
// Larger file size correlates with more visual information (less flat areas).
// Returns empty string if no file exists.
func selectBestFrame(paths []string) string {
	best := ""
	var bestSize int64
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.Size() > bestSize {
			bestSize = info.Size()
			best = p
		}
	}
	return best
}
