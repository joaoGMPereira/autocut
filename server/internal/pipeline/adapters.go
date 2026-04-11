package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/processor"
	"github.com/joaoGMPereira/autocut/server/internal/thumbnail"
	"github.com/joaoGMPereira/autocut/server/internal/transcript"
	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

// ── YouTubeDownloaderAdapter ──────────────────────────────────────────────────

// YouTubeDownloaderAdapter wraps *downloader.YouTubeDownloader for the pipeline Downloader interface.
type YouTubeDownloaderAdapter struct {
	dl *downloader.YouTubeDownloader
}

// NewYouTubeDownloaderAdapter creates a Downloader adapter.
func NewYouTubeDownloaderAdapter(dl *downloader.YouTubeDownloader) Downloader {
	return &YouTubeDownloaderAdapter{dl: dl}
}

// DownloadVideo implements Downloader.
func (a *YouTubeDownloaderAdapter) DownloadVideo(ctx context.Context, url, outputDir string) (string, error) {
	// Local file pass-through
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "~/") {
		expanded := url
		if strings.HasPrefix(url, "~/") {
			home, _ := os.UserHomeDir()
			expanded = filepath.Join(home, url[2:])
		}
		if _, err := os.Stat(expanded); err == nil {
			return expanded, nil
		}
		return "", fmt.Errorf("local file not found: %s", expanded)
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	info, err := a.dl.DownloadWithOptions(url, outputDir, "")
	if err != nil {
		return "", fmt.Errorf("youtube download: %w", err)
	}
	if info == nil {
		return "", fmt.Errorf("youtube download returned nil info")
	}
	return info.FilePath, nil
}

// ── FFmpegCutterAdapter ───────────────────────────────────────────────────────

// FFmpegCutterAdapter wraps *processor.FFmpegProcessor for the pipeline Cutter interface.
type FFmpegCutterAdapter struct {
	ffmpeg *processor.FFmpegProcessor
}

// NewFFmpegCutterAdapter creates a Cutter adapter.
func NewFFmpegCutterAdapter(ffmpeg *processor.FFmpegProcessor) Cutter {
	return &FFmpegCutterAdapter{ffmpeg: ffmpeg}
}

// CutSegments implements Cutter.
func (a *FFmpegCutterAdapter) CutSegments(ctx context.Context, sourcePath string, segments []SegmentRange, outputDir string) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("cut: create output dir: %w", err)
	}
	if len(segments) == 0 {
		return []string{sourcePath}, nil
	}
	var clips []string
	for i, seg := range segments {
		select {
		case <-ctx.Done():
			return clips, ctx.Err()
		default:
		}
		if seg.EndSec <= seg.StartSec {
			continue
		}
		start := time.Duration(seg.StartSec * float64(time.Second))
		end := time.Duration(seg.EndSec * float64(time.Second))
		ext := filepath.Ext(sourcePath)
		if ext == "" {
			ext = ".mp4"
		}
		outPath := filepath.Join(outputDir, fmt.Sprintf("clip_%03d%s", i, ext))
		if err := a.ffmpeg.CutClip(sourcePath, start, end, outPath); err != nil {
			return nil, fmt.Errorf("cut: clip %d: %w", i, err)
		}
		clips = append(clips, outPath)
	}
	return clips, nil
}

// ── WhisperTranscriberAdapter ─────────────────────────────────────────────────

// WhisperTranscriberAdapter wraps *transcript.WhisperTranscriber for the pipeline Transcriber interface.
type WhisperTranscriberAdapter struct {
	whisper  *transcript.WhisperTranscriber
	cache    *transcript.TranscriptCache
	cacheDir string
}

// NewWhisperTranscriberAdapter creates a Transcriber adapter.
func NewWhisperTranscriberAdapter(w *transcript.WhisperTranscriber, c *transcript.TranscriptCache, cacheDir string) Transcriber {
	return &WhisperTranscriberAdapter{whisper: w, cache: c, cacheDir: cacheDir}
}

// Transcribe implements Transcriber.
func (a *WhisperTranscriberAdapter) Transcribe(ctx context.Context, sourcePath string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	hash, hashErr := a.cache.Hash(sourcePath)
	if hashErr == nil {
		if cached, ok := a.cache.Get(hash); ok {
			return a.saveTranscriptJSON(hash, cached)
		}
	}
	t, err := a.whisper.Transcribe(sourcePath)
	if err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	if hashErr == nil {
		_ = a.cache.Put(hash, t)
	}
	key := hash
	if key == "" {
		key = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	return a.saveTranscriptJSON(key, t)
}

func (a *WhisperTranscriberAdapter) saveTranscriptJSON(key string, t *transcript.Transcript) (string, error) {
	if err := os.MkdirAll(a.cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("save transcript: mkdir: %w", err)
	}
	outPath := filepath.Join(a.cacheDir, key+".json")
	b, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("save transcript: marshal: %w", err)
	}
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		return "", fmt.Errorf("save transcript: write: %w", err)
	}
	return outPath, nil
}

// ── transcriptSegmentAdapter ──────────────────────────────────────────────────

type transcriptSegmentAdapter struct{ seg transcript.Segment }

func (a transcriptSegmentAdapter) GetStart() float64 { return a.seg.Start.Seconds() }
func (a transcriptSegmentAdapter) GetEnd() float64   { return a.seg.End.Seconds() }
func (a transcriptSegmentAdapter) GetText() string   { return a.seg.Text }

// ── AIAnalyzerAdapter ─────────────────────────────────────────────────────────

// AIAnalyzerAdapter wraps *ai.TopicTransitionDetector for the pipeline Analyzer interface.
type AIAnalyzerAdapter struct {
	detector *ai.TopicTransitionDetector
}

// NewAIAnalyzerAdapter creates an Analyzer adapter.
func NewAIAnalyzerAdapter(detector *ai.TopicTransitionDetector) Analyzer {
	return &AIAnalyzerAdapter{detector: detector}
}

// Analyze implements Analyzer.
func (a *AIAnalyzerAdapter) Analyze(ctx context.Context, transcriptPath string) ([]Highlight, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		return nil, fmt.Errorf("analyze: read transcript: %w", err)
	}
	var t transcript.Transcript
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("analyze: unmarshal transcript: %w", err)
	}
	segs := make([]ai.TranscriptSegment, len(t.Segments))
	for i, s := range t.Segments {
		segs[i] = transcriptSegmentAdapter{seg: s}
	}
	result, err := a.detector.Analyze(ctx, segs)
	if err != nil {
		return nil, fmt.Errorf("analyze: detector: %w", err)
	}
	out := make([]Highlight, len(result.Highlights))
	for i, h := range result.Highlights {
		out[i] = Highlight{
			StartSec:    h.StartSec,
			EndSec:      h.EndSec,
			AdjStartSec: h.StartSec,
			AdjEndSec:   h.EndSec,
			Score:       h.Score,
			Reason:      h.Reason,
			IsSelected:  true,
		}
	}
	return out, nil
}

// ── ShortsGeneratorAdapter ────────────────────────────────────────────────────

// ShortsGeneratorAdapter wraps *processor.ShortsGenerator for the pipeline ShortsGenerator interface.
type ShortsGeneratorAdapter struct {
	gen    *processor.ShortsGenerator
	ffmpeg *processor.FFmpegProcessor
}

// NewShortsGeneratorAdapter creates a ShortsGenerator adapter.
func NewShortsGeneratorAdapter(gen *processor.ShortsGenerator, ffmpeg *processor.FFmpegProcessor) ShortsGenerator {
	return &ShortsGeneratorAdapter{gen: gen, ffmpeg: ffmpeg}
}

// GenerateShorts implements ShortsGenerator.
func (a *ShortsGeneratorAdapter) GenerateShorts(ctx context.Context, sourcePath string, segments []SegmentRange, outputDir string) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("shorts: create output dir: %w", err)
	}
	if len(segments) == 0 {
		return nil, nil
	}
	const maxShortDurationSec = 120.0

	logger := slog.Default()
	var paths []string
	for i, seg := range segments {
		select {
		case <-ctx.Done():
			return paths, ctx.Err()
		default:
		}
		if seg.EndSec <= seg.StartSec {
			continue
		}
		duration := seg.EndSec - seg.StartSec
		if duration > maxShortDurationSec {
			logger.Warn("skipping short for segment exceeding max duration",
				"segment", i, "duration_sec", duration, "max_sec", maxShortDurationSec)
			continue
		}
		start := time.Duration(seg.StartSec * float64(time.Second))
		end := time.Duration(seg.EndSec * float64(time.Second))
		tmpPath := filepath.Join(outputDir, fmt.Sprintf("short_tmp_%03d.mp4", i))
		if err := a.ffmpeg.CutClip(sourcePath, start, end, tmpPath); err != nil {
			return nil, fmt.Errorf("shorts: cut clip %d: %w", i, err)
		}
		outPath := filepath.Join(outputDir, fmt.Sprintf("short_%03d.mp4", i))
		cfg := processor.ShortsConfig{
			Width:             1080,
			Height:            1920,
			Language:          "pt",
			AddBlurBackground: true,
		}
		genErr := a.gen.Generate(tmpPath, cfg, outPath)
		// Remove the tmp cut file immediately after generate (success or failure)
		// so it doesn't accumulate across iterations — defer inside a loop only
		// runs at function return, not per-iteration.
		os.Remove(tmpPath)
		if genErr != nil {
			return nil, fmt.Errorf("shorts: generate %d: %w", i, genErr)
		}
		paths = append(paths, outPath)
	}
	return paths, nil
}

// ── ThumbnailMakerAdapter ─────────────────────────────────────────────────────

// ThumbnailMakerAdapter wraps *thumbnail.ThumbnailGenerator for the pipeline ThumbnailMaker interface.
type ThumbnailMakerAdapter struct {
	gen *thumbnail.ThumbnailGenerator
	log *slog.Logger
}

// NewThumbnailMakerAdapter creates a ThumbnailMaker adapter.
func NewThumbnailMakerAdapter(gen *thumbnail.ThumbnailGenerator) ThumbnailMaker {
	return &ThumbnailMakerAdapter{gen: gen, log: slog.With("component", "pipeline.thumbnail")}
}

// MakeThumbnail implements ThumbnailMaker.
func (a *ThumbnailMakerAdapter) MakeThumbnail(ctx context.Context, sourcePath, style string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	outPath := sourcePath[:len(sourcePath)-len(filepath.Ext(sourcePath))] + "_thumb.jpg"
	cfg := thumbnail.LongFormConfig{
		VideoPath: sourcePath,
		FontColor: "#FFFFFF",
	}
	if err := a.gen.GenerateLongForm(cfg, outPath); err != nil {
		return "", fmt.Errorf("thumbnail: %w", err)
	}
	return outPath, nil
}

// ── UploaderAdapter ───────────────────────────────────────────────────────────

// UploaderAdapter wraps *uploader.YouTubeUploader for the pipeline Uploader interface.
type UploaderAdapter struct {
	ul  *uploader.YouTubeUploader
	log *slog.Logger
}

// NewUploaderAdapter creates an Uploader adapter.
func NewUploaderAdapter(ul *uploader.YouTubeUploader) Uploader {
	return &UploaderAdapter{ul: ul, log: slog.With("component", "pipeline.uploader")}
}

// UploadClip implements Uploader.
func (a *UploaderAdapter) UploadClip(ctx context.Context, req UploadClipRequest) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	privacy := req.Privacy
	if privacy == "" {
		privacy = "private"
	}
	uReq := uploader.UploadRequest{
		FilePath:    req.FilePath,
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
		Privacy:     privacy,
	}
	ch, err := a.ul.Upload(ctx, uReq)
	if err != nil {
		return "", fmt.Errorf("upload: %w", err)
	}
	var youtubeID string
	for prog := range ch {
		if prog.Done {
			if prog.Err != nil {
				return "", fmt.Errorf("upload: %w", prog.Err)
			}
			youtubeID = prog.VideoID
		}
	}
	return youtubeID, nil
}
