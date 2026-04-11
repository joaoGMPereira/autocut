package thumbnail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
// GenerateTemplate
// ---------------------------------------------------------------------------

// TemplateConfig holds parameters for a template-strategy thumbnail.
// The background image is loaded from a per-channel stored background (thumbnail_backgrounds table).
// Kotlin ref: TemplateStrategy config (FP-007)
type TemplateConfig struct {
	// BackgroundPath is the absolute path to the stored background image.
	BackgroundPath string
	// Title is the text drawn centred over the background.
	Title string
	// FontSize in points. Default 36.
	FontSize int
	// FontColor is an FFmpeg colour string. Default "white".
	FontColor string
	// Width of the output image. Default 1280.
	Width int
	// Height of the output image. Default 720.
	Height int
}

// withDefaults returns a copy of c with zero-valued fields filled in.
func (c TemplateConfig) withDefaults() TemplateConfig {
	if c.FontSize == 0 {
		c.FontSize = 36
	}
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	if c.Width == 0 {
		c.Width = 1280
	}
	if c.Height == 0 {
		c.Height = 720
	}
	return c
}

// GenerateTemplate generates a thumbnail from a stored background image with a
// centred text overlay using the FFmpeg drawtext filter.
//
// FFmpeg command equivalent:
//
//	ffmpeg -i {bg} -vf "scale=W:H:force_original_aspect_ratio=increase,crop=W:H,
//	  drawtext=text='{title}':fontsize=36:fontcolor=white:x=(w-tw)/2:y=(h-th)/2"
//	  -frames:v 1 {output}
//
// Kotlin ref: TemplateStrategy.generate (FP-007)
func (g *ThumbnailGenerator) GenerateTemplate(cfg TemplateConfig, output string) error {
	cfg = cfg.withDefaults()

	title := escapeDrawtext(cfg.Title)

	filter := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,"+
			"drawtext=text='%s':fontsize=%d:fontcolor=%s:x=(w-tw)/2:y=(h-th)/2",
		cfg.Width, cfg.Height, cfg.Width, cfg.Height,
		title, cfg.FontSize, cfg.FontColor,
	)

	args := []string{
		"-y",
		"-i", cfg.BackgroundPath,
		"-vf", filter,
		"-frames:v", "1",
		"-q:v", "2",
		output,
	}

	g.log.Debug("GenerateTemplate", "op", "template",
		"bg", cfg.BackgroundPath, "output", output)

	if out, err := g.exec.Run(g.ffmpegPath, args...); err != nil {
		return fmt.Errorf("GenerateTemplate: ffmpeg: %w\noutput: %s", err, out)
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

// ---------------------------------------------------------------------------
// GenerateAIThumbnail
// ---------------------------------------------------------------------------

// AIThumbnailConfig holds parameters for an AI-generated thumbnail.
// The Ollama vision model (llava) generates a CTR-optimised title from a
// video frame; the result is rendered via the branded strategy.
type AIThumbnailConfig struct {
	// VideoPath is the source video file.
	VideoPath string
	// CustomPrompt overrides the default Ollama prompt when non-empty.
	CustomPrompt string
	// OllamaURL is the base URL of the Ollama HTTP server.
	// Default: "http://localhost:11434" or OLLAMA_URL env var.
	OllamaURL string
	// FontPath is an optional absolute path to a .ttf/.otf font file.
	FontPath string
	// FontColor is an FFmpeg colour string. Default "white".
	FontColor string
	// FontSize in points. Default 80.
	FontSize int
	// Width of the output image. Default 1280.
	Width int
	// Height of the output image. Default 720.
	Height int
}

// withDefaults returns a copy of c with zero-valued fields filled in.
func (c AIThumbnailConfig) withDefaults() AIThumbnailConfig {
	if c.OllamaURL == "" {
		if u := os.Getenv("OLLAMA_URL"); u != "" {
			c.OllamaURL = u
		} else {
			c.OllamaURL = "http://localhost:11434"
		}
	}
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	if c.FontSize == 0 {
		c.FontSize = 80
	}
	if c.Width == 0 {
		c.Width = 1280
	}
	if c.Height == 0 {
		c.Height = 720
	}
	return c
}

// ollamaVisionRequest is the JSON payload for Ollama's /api/generate endpoint
// when using a vision model (llava). Images must be base64-encoded strings.
type ollamaVisionRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images,omitempty"` // base64-encoded image bytes
	Stream bool     `json:"stream"`
}

// ollamaVisionResponse is the non-streaming JSON response from Ollama.
type ollamaVisionResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

const defaultAIPrompt = "You are a YouTube thumbnail text generator. " +
	"Look at this video frame and generate a short, punchy title for a YouTube thumbnail. " +
	"Focus on what would get the most clicks. " +
	"Respond with ONLY the title text, nothing else, max 6 words."

// GenerateAIThumbnail uses Ollama to generate an AI-enhanced thumbnail.
// It extracts a frame at 30 % of video duration, sends it to Ollama (llava)
// for a CTR-optimised title, then renders a branded thumbnail with that title.
// Falls back gracefully if Ollama is unavailable.
//
// Kotlin ref: (new — FP-008) AIThumbnailStrategy
func (g *ThumbnailGenerator) GenerateAIThumbnail(cfg AIThumbnailConfig, output string) error {
	cfg = cfg.withDefaults()

	// Step 1: extract a frame at 30 % of video duration.
	dur, err := g.getVideoDuration(cfg.VideoPath)
	if err != nil {
		g.log.Warn("GenerateAIThumbnail: duration detection failed, using 10s fallback",
			"op", "ai_thumbnail", "video", cfg.VideoPath, "err", err)
		dur = 10 * time.Second
	}

	ts := time.Duration(float64(dur) * 0.3)
	framePath := filepath.Join(os.TempDir(), fmt.Sprintf("thumb_ai_frame_%d.jpg", time.Now().UnixNano()))
	defer func() {
		if removeErr := os.Remove(framePath); removeErr != nil && !os.IsNotExist(removeErr) {
			g.log.Warn("GenerateAIThumbnail: could not remove temp frame",
				"op", "ai_thumbnail", "path", framePath, "err", removeErr)
		}
	}()

	if extractErr := g.extractFrame(cfg.VideoPath, ts, framePath); extractErr != nil {
		g.log.Warn("GenerateAIThumbnail: frame extraction failed, continuing without image",
			"op", "ai_thumbnail", "err", extractErr)
		framePath = "" // will skip image encoding below
	}

	// Step 2: build Ollama prompt.
	prompt := cfg.CustomPrompt
	if strings.TrimSpace(prompt) == "" {
		prompt = defaultAIPrompt
	}

	// Step 3: call Ollama; derive overlay text from response.
	aiText := g.callOllamaForTitle(cfg.OllamaURL, prompt, framePath)

	g.log.Info("GenerateAIThumbnail: AI text generated",
		"op", "ai_thumbnail", "text", aiText)

	// Step 4: render branded thumbnail with AI-generated text.
	brandedCfg := BrandedConfig{
		BackgroundPath: cfg.VideoPath,
		TextOverlay:    aiText,
		FontPath:       cfg.FontPath,
		FontColor:      cfg.FontColor,
		FontSize:       cfg.FontSize,
		Width:          cfg.Width,
		Height:         cfg.Height,
		Position:       Position{Anchor: AnchorCenter},
	}
	return g.GenerateBranded(brandedCfg, output)
}

// callOllamaForTitle sends a generate request to Ollama and returns the
// trimmed response string. Returns a safe fallback string on any error.
func (g *ThumbnailGenerator) callOllamaForTitle(ollamaURL, prompt, framePath string) string {
	req := ollamaVisionRequest{
		Model:  "llava",
		Prompt: prompt,
		Stream: false,
	}

	// Encode frame image when available; fall back to llama3 text-only if not.
	if framePath != "" {
		imageBytes, readErr := os.ReadFile(framePath)
		if readErr == nil {
			req.Images = []string{base64.StdEncoding.EncodeToString(imageBytes)}
		} else {
			g.log.Warn("callOllamaForTitle: could not read frame image, using text-only model",
				"op", "ai_thumbnail", "err", readErr)
			req.Model = "llama3"
		}
	} else {
		req.Model = "llama3"
	}

	body, marshalErr := json.Marshal(req)
	if marshalErr != nil {
		g.log.Warn("callOllamaForTitle: marshal failed", "op", "ai_thumbnail", "err", marshalErr)
		return "Epic Moment"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	url := strings.TrimRight(ollamaURL, "/") + "/api/generate"
	httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if reqErr != nil {
		g.log.Warn("callOllamaForTitle: build request failed", "op", "ai_thumbnail", "err", reqErr)
		return "Epic Moment"
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, doErr := http.DefaultClient.Do(httpReq)
	if doErr != nil {
		g.log.Warn("callOllamaForTitle: Ollama unavailable", "op", "ai_thumbnail", "err", doErr)
		return "Epic Moment"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		g.log.Warn("callOllamaForTitle: unexpected status",
			"op", "ai_thumbnail", "status", resp.StatusCode)
		return "Epic Moment"
	}

	var ollamaResp ollamaVisionResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&ollamaResp); decodeErr != nil {
		g.log.Warn("callOllamaForTitle: decode failed", "op", "ai_thumbnail", "err", decodeErr)
		return "Epic Moment"
	}

	text := strings.TrimSpace(ollamaResp.Response)
	if text == "" {
		return "Epic Moment"
	}
	return text
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
