package processor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
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
	// AddBlurBackground enables the 9:16 blur composite instead of simple crop.
	// Kotlin ref: LL-005 — ShortsConfig.AddBlurBackground bool (previously reserved).
	// When true, the original content is centred on a blurred-and-cropped background.
	AddBlurBackground bool
	// Language is the BCP-47 language tag for captions and keyword scoring.
	// Defaults to "pt". Used by engagement scorer and subtitle style.
	Language string
}

func (c *ShortsConfig) setDefaults() {
	if c.Width == 0 {
		c.Width = 1080
	}
	if c.Height == 0 {
		c.Height = 1920
	}
	if c.Language == "" {
		c.Language = "pt"
	}
}

// ShortsGenerator converts horizontal videos to vertical 9:16 Shorts format.
// Kotlin ref: ShortsGenerator + VerticalVideoProcessor.convertToVertical.
type ShortsGenerator struct {
	ffmpeg   *FFmpegProcessor
	subtitles *SubtitleGenerator
	log      *slog.Logger
}

// NewShortsGenerator creates a ShortsGenerator.
// ffmpegPath defaults to "ffmpeg" when empty.
func NewShortsGenerator(ffmpegPath string) *ShortsGenerator {
	return &ShortsGenerator{
		ffmpeg:   NewFFmpegProcessor(ffmpegPath),
		subtitles: NewSubtitleGenerator(ffmpegPath),
		log:      slog.With("component", "processor", "tool", "shorts"),
	}
}

// newShortsGeneratorWithExecutor injects a mock Executor for tests.
func newShortsGeneratorWithExecutor(ffmpegPath string, ex Executor) *ShortsGenerator {
	fp := newFFmpegProcessorWithExecutor(ffmpegPath, ex)
	sg := newSubtitleGeneratorWithExecutor(ffmpegPath, ex)
	return &ShortsGenerator{
		ffmpeg:   fp,
		subtitles: sg,
		log:      slog.With("component", "processor", "tool", "shorts"),
	}
}

// Generate converts input to a vertical short using cfg, writing to output.
// Dispatches to generateBlur when cfg.AddBlurBackground=true, otherwise uses
// the original crop+scale path.
//
// Kotlin ref: VerticalVideoProcessor.buildFfmpegCommand + buildVerticalFilter.
func (g *ShortsGenerator) Generate(input string, cfg ShortsConfig, output string) error {
	cfg.setDefaults()
	if cfg.AddBlurBackground {
		return g.generateBlur(input, cfg, output)
	}
	return g.generateCrop(input, cfg, output)
}

// generateCrop is the original crop+scale path (backward compatible).
// crop=w:h:x:y then scale to final dimensions.
func (g *ShortsGenerator) generateCrop(input string, cfg ShortsConfig, output string) error {
	_ = os.MkdirAll(filepath.Dir(output), 0o755)

	vf := fmt.Sprintf("crop=%d:%d:%d:%d,scale=%d:%d",
		cfg.Width, cfg.Height, cfg.CropX, cfg.CropY,
		cfg.Width, cfg.Height,
	)

	args := []string{
		"-loglevel", "error",
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

	g.log.Debug("generateCrop", "op", "shorts", "input", input, "vf", vf)
	if _, err := g.ffmpeg.exec.Run(g.ffmpeg.binPath, args...); err != nil {
		g.log.Error("generateCrop failed", "err", err, "input", input)
		return fmt.Errorf("shorts generate (crop): %w", err)
	}
	return nil
}

// generateBlur produces a 9:16 vertical video with the original content centred
// on a blurred copy of itself at full 9:16 resolution.
//
// FFmpeg filter chain (LL-005 implementation):
//
//	[0:v]scale=W:H:force_original_aspect_ratio=increase,crop=W:H,boxblur=20:20[bg]
//	[0:v]scale=iw*min(W/iw\,H/ih):ih*min(W/iw\,H/ih)[scaled]
//	[bg][scaled]overlay=(W-w)/2:(H-h)/2[v]
//
// Kotlin ref: VerticalVideoProcessor.buildBlurCompositeFilter (LL-005 deferred).
func (g *ShortsGenerator) generateBlur(input string, cfg ShortsConfig, output string) error {
	_ = os.MkdirAll(filepath.Dir(output), 0o755)

	W := cfg.Width
	H := cfg.Height

	bgFilter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,boxblur=20:20[bg]",
		W, H, W, H,
	)
	scaledFilter := fmt.Sprintf(
		"[0:v]scale=iw*min(%d/iw\\,%d/ih):ih*min(%d/iw\\,%d/ih)[scaled]",
		W, H, W, H,
	)
	overlayFilter := "[bg][scaled]overlay=(W-w)/2:(H-h)/2[v]"

	filterComplex := strings.Join([]string{bgFilter, scaledFilter, overlayFilter}, ";")

	args := []string{
		"-loglevel", "error",
		"-i", input,
		"-filter_complex", filterComplex,
		"-map", "[v]",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-movflags", "+faststart",
		"-y",
		output,
	}

	g.log.Debug("generateBlur", "op", "shorts_blur", "input", input,
		"w", W, "h", H)
	if _, err := g.ffmpeg.exec.Run(g.ffmpeg.binPath, args...); err != nil {
		g.log.Error("generateBlur failed", "err", err, "input", input)
		return fmt.Errorf("shorts generate (blur): %w", err)
	}
	return nil
}

// GenerateCaptionedShort produces a vertical short with word-by-word captions.
//
// Pipeline:
//  1. Generate vertical video to a temp file (blur or crop depending on cfg)
//  2. Split segments into per-word SubtitleSegments
//  3. Burn captions via SubtitleGenerator.GenerateAndBurn
//  4. Cleanup temp file
//
// Kotlin ref: ShortsOllamaDelegate.generateCaptioned (caption burn-in step).
func (g *ShortsGenerator) GenerateCaptionedShort(
	input string,
	segments []SubtitleSegment,
	output string,
	cfg ShortsConfig,
) error {
	cfg.setDefaults()

	tmpFile := filepath.Join(os.TempDir(),
		fmt.Sprintf("autocut_shorts_cap_%d.mp4", time.Now().UnixNano()))
	defer func() {
		if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
			g.log.Warn("GenerateCaptionedShort: cleanup failed", "tmp", tmpFile, "err", err)
		}
	}()

	// Step 1 — vertical conversion
	if err := g.Generate(input, cfg, tmpFile); err != nil {
		g.log.Error("GenerateCaptionedShort: vertical conversion failed",
			"err", err, "input", input)
		return fmt.Errorf("shorts captioned: vertical convert: %w", err)
	}

	// Step 2 — split into word-level segments
	wordSegs := transcriptToWordSegments(segments)
	g.log.Debug("GenerateCaptionedShort", "op", "captions",
		"input_segments", len(segments), "word_segments", len(wordSegs))

	// Step 3 — burn captions
	captionCfg := SubtitleConfig{
		FontName:  "Arial",
		FontSize:  24,
		FontColor: "white",
		Outline:   true,
		Position:  "bottom",
	}
	if err := g.subtitles.GenerateAndBurn(tmpFile, wordSegs, output, captionCfg); err != nil {
		g.log.Error("GenerateCaptionedShort: burn failed", "err", err)
		return fmt.Errorf("shorts captioned: burn: %w", err)
	}
	return nil
}

// transcriptToWordSegments splits each SubtitleSegment into one segment per word,
// distributing the source segment's time range proportionally across the words.
//
// Example: segment "hello world" spanning [0s, 1s] → two segments:
//   - [0s, 0.5s] "hello"
//   - [0.5s, 1s] "world"
//
// Segments with no words or zero duration are skipped.
func transcriptToWordSegments(segments []SubtitleSegment) []SubtitleSegment {
	var result []SubtitleSegment
	for _, seg := range segments {
		words := splitWords(seg.Text)
		if len(words) == 0 {
			continue
		}
		dur := seg.End - seg.Start
		if dur <= 0 {
			// Emit the whole segment as-is when duration is zero/negative.
			result = append(result, seg)
			continue
		}
		wordDur := dur / time.Duration(len(words))
		for i, w := range words {
			start := seg.Start + time.Duration(i)*wordDur
			end := start + wordDur
			if i == len(words)-1 {
				end = seg.End // last word gets any rounding remainder
			}
			result = append(result, SubtitleSegment{
				Start: start,
				End:   end,
				Text:  w,
			})
		}
	}
	return result
}

// splitWords splits text on whitespace, filtering empty tokens.
func splitWords(text string) []string {
	var words []string
	start := -1
	for i, r := range text {
		if unicode.IsSpace(r) {
			if start >= 0 {
				words = append(words, text[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, text[start:])
	}
	return words
}
