package processor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SubtitleSegment is a single timed subtitle entry.
// Kotlin ref: SubtitleGenerator.SubtitleEntry (start/end + text)
type SubtitleSegment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

// SubtitleConfig controls how subtitles are rendered when burned into video.
// Kotlin ref: SubtitleStyle / SubtitleConfig in SubtitleBurnIn
type SubtitleConfig struct {
	FontName  string // default "Arial"
	FontSize  int    // default 20
	FontColor string // default "white"
	Outline   bool   // default true
	Position  string // default "bottom" (top|bottom|center)
}

// SubtitleGenerator creates SRT files and burns subtitles into video via FFmpeg.
// Kotlin ref: SubtitleGenerator + SubtitleBurnIn (merged — simpler surface area)
type SubtitleGenerator struct {
	ffmpegPath string
	exec       Executor
	log        *slog.Logger
}

// NewSubtitleGenerator creates a SubtitleGenerator.
// ffmpegPath defaults to "ffmpeg" when empty.
func NewSubtitleGenerator(ffmpegPath string) *SubtitleGenerator {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &SubtitleGenerator{
		ffmpegPath: ffmpegPath,
		exec:       &DefaultExecutor{},
		log:        slog.With("component", "subtitle_generator"),
	}
}

// newSubtitleGeneratorWithExecutor injects a mock executor for tests.
func newSubtitleGeneratorWithExecutor(ffmpegPath string, ex Executor) *SubtitleGenerator {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &SubtitleGenerator{
		ffmpegPath: ffmpegPath,
		exec:       ex,
		log:        slog.With("component", "subtitle_generator"),
	}
}

// GenerateSRT writes an SRT subtitle file to outputPath.
//
// SRT format:
//
//	1
//	00:00:00,000 --> 00:00:01,500
//	Hello world
//
//	2
//	...
//
// Kotlin ref: SubtitleGenerator.generateSRT
func (g *SubtitleGenerator) GenerateSRT(segments []SubtitleSegment, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("generate srt: mkdir: %w", err)
	}

	var sb strings.Builder
	for i, seg := range segments {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "%d\n", i+1)
		fmt.Fprintf(&sb, "%s --> %s\n", formatSRTTimestamp(seg.Start), formatSRTTimestamp(seg.End))
		sb.WriteString(seg.Text)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("generate srt: write: %w", err)
	}
	g.log.Debug("GenerateSRT", "op", "srt", "segments", len(segments), "output", outputPath)
	return nil
}

// BurnSubtitles runs FFmpeg to burn an existing SRT file into a video.
//
// FFmpeg filter:
//
//	subtitles={srtPath}:force_style='FontName={font},FontSize={size},PrimaryColour=&H{color}&,Outline={outline}'
//
// Kotlin ref: SubtitleBurnIn.burn
func (g *SubtitleGenerator) BurnSubtitles(videoPath, srtPath, output string, cfg SubtitleConfig) error {
	cfg = defaultSubtitleConfig(cfg)

	outline := "0"
	if cfg.Outline {
		outline = "1"
	}

	style := fmt.Sprintf(
		"FontName=%s,FontSize=%d,PrimaryColour=&H%s&,Outline=%s",
		cfg.FontName, cfg.FontSize, colorToHex(cfg.FontColor), outline,
	)

	// Apply vertical alignment via MarginV when position is set.
	switch cfg.Position {
	case "top":
		style += ",Alignment=8"
	case "center":
		style += ",Alignment=5"
	default: // "bottom"
		style += ",Alignment=2"
	}

	vf := fmt.Sprintf("subtitles=%s:force_style='%s'", srtPath, style)

	args := []string{
		"-i", videoPath,
		"-vf", vf,
		"-c:a", "copy",
		"-y",
		output,
	}

	g.log.Debug("BurnSubtitles", "op", "burn", "video", videoPath, "srt", srtPath, "output", output)
	if _, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		g.log.Error("BurnSubtitles failed", "err", err, "video", videoPath)
		return fmt.Errorf("burn subtitles: %w", err)
	}
	return nil
}

// GenerateAndBurn creates a temporary SRT file, burns it into videoPath, then
// removes the temp file. Convenience wrapper around GenerateSRT + BurnSubtitles.
//
// Kotlin ref: SubtitleGenerator.generateAndBurn (pipeline method)
func (g *SubtitleGenerator) GenerateAndBurn(
	videoPath string,
	segments []SubtitleSegment,
	output string,
	cfg SubtitleConfig,
) error {
	tmpSRT := filepath.Join(os.TempDir(), fmt.Sprintf("autocut_subs_%d.srt", time.Now().UnixNano()))
	if err := g.GenerateSRT(segments, tmpSRT); err != nil {
		return fmt.Errorf("generate and burn: srt: %w", err)
	}
	defer os.Remove(tmpSRT)

	if err := g.BurnSubtitles(videoPath, tmpSRT, output, cfg); err != nil {
		return fmt.Errorf("generate and burn: burn: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// formatSRTTimestamp converts a Duration to SRT timestamp format "HH:MM:SS,mmm".
// Example: 3661500ms → "01:01:01,500"
func formatSRTTimestamp(d time.Duration) string {
	total := d.Milliseconds()
	ms := total % 1000
	total /= 1000
	s := total % 60
	total /= 60
	m := total % 60
	h := total / 60
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// defaultSubtitleConfig returns a copy of cfg with zero-valued fields filled.
func defaultSubtitleConfig(cfg SubtitleConfig) SubtitleConfig {
	if cfg.FontName == "" {
		cfg.FontName = "Arial"
	}
	if cfg.FontSize == 0 {
		cfg.FontSize = 20
	}
	if cfg.FontColor == "" {
		cfg.FontColor = "white"
	}
	if cfg.Position == "" {
		cfg.Position = "bottom"
	}
	return cfg
}

// colorToHex converts a named color or hex string to an ASS-compatible hex.
// Only the most common names are mapped; unknown values are passed through.
func colorToHex(color string) string {
	switch strings.ToLower(color) {
	case "white":
		return "FFFFFF"
	case "black":
		return "000000"
	case "yellow":
		return "00FFFF"
	case "red":
		return "0000FF"
	case "blue":
		return "FF0000"
	case "green":
		return "00FF00"
	default:
		// Already a hex string — return as-is
		return strings.TrimPrefix(color, "#")
	}
}
