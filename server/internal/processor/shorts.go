package processor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// ShortsConfig holds parameters for vertical (9:16) short-form video generation.
// Kotlin ref: ShortsConfig model + VerticalVideoProcessor constants.
type ShortsConfig struct {
	// Width of the output video. Defaults to 1080.
	Width int
	// Height of the output video. Defaults to 1920.
	Height int
	// CropX is the horizontal crop offset applied before scaling. 0 = centre.
	CropX int
	// CropY is the vertical crop offset applied before scaling. 0 = centre.
	CropY int
	// AddSubtitles enables subtitle burn-in (future use — requires subtitle file).
	AddSubtitles bool
}

func (c *ShortsConfig) setDefaults() {
	if c.Width == 0 {
		c.Width = 1080
	}
	if c.Height == 0 {
		c.Height = 1920
	}
}

// ShortsGenerator converts horizontal videos to vertical 9:16 Shorts format.
// Kotlin ref: ShortsGenerator + VerticalVideoProcessor.convertToVertical.
type ShortsGenerator struct {
	ffmpeg *FFmpegProcessor
	log    *slog.Logger
}

// NewShortsGenerator creates a ShortsGenerator.
// ffmpegPath defaults to "ffmpeg" when empty.
func NewShortsGenerator(ffmpegPath string) *ShortsGenerator {
	return &ShortsGenerator{
		ffmpeg: NewFFmpegProcessor(ffmpegPath),
		log:    slog.With("component", "processor", "tool", "shorts"),
	}
}

// newShortsGeneratorWithExecutor injects a mock Executor for tests.
func newShortsGeneratorWithExecutor(ffmpegPath string, ex Executor) *ShortsGenerator {
	return &ShortsGenerator{
		ffmpeg: newFFmpegProcessorWithExecutor(ffmpegPath, ex),
		log:    slog.With("component", "processor", "tool", "shorts"),
	}
}

// Generate converts input to a vertical short using cfg, writing to output.
//
// The crop filter crops to cfg.Width x cfg.Height starting at (cfg.CropX, cfg.CropY),
// then scales to 1080x1920 — matching VerticalVideoProcessor dimensions.
//
// Kotlin ref: VerticalVideoProcessor.buildFfmpegCommand + buildVerticalFilter.
func (g *ShortsGenerator) Generate(input string, cfg ShortsConfig, output string) error {
	cfg.setDefaults()

	_ = os.MkdirAll(filepath.Dir(output), 0o755)

	// crop=w:h:x:y then scale to final dimensions.
	vf := fmt.Sprintf("crop=%d:%d:%d:%d,scale=%d:%d",
		cfg.Width, cfg.Height, cfg.CropX, cfg.CropY,
		cfg.Width, cfg.Height,
	)

	args := []string{
		"-i", input,
		"-vf", vf,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-movflags", "+faststart",
		"-y",
		output,
	}

	g.log.Debug("Generate", "op", "shorts", "input", input, "vf", vf)
	if _, err := g.ffmpeg.exec.Run(g.ffmpeg.binPath, args...); err != nil {
		g.log.Error("Generate failed", "err", err, "input", input)
		return fmt.Errorf("shorts generate: %w", err)
	}
	return nil
}
