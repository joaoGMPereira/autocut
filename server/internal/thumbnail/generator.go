package thumbnail

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// thumbnailProgressPayload is the SSE payload for thumbnail_progress events.
type thumbnailProgressPayload struct {
	RunID         int64  `json:"run_id"`
	ClipID        int64  `json:"clip_id"`
	ClipIndex     int    `json:"clip_index"`
	TotalClips    int    `json:"total_clips"`
	Status        string `json:"status"`
	Strategy      string `json:"strategy,omitempty"`
	ThumbnailPath string `json:"thumbnail_path,omitempty"`
	Error         string `json:"error,omitempty"`
	SuccessCount  int    `json:"success_count,omitempty"`
	ErrorCount    int    `json:"error_count,omitempty"`
}

// Generator orchestrates thumbnail generation for clips.
type Generator struct {
	clipRepo      *database.PipelineClipRepo
	hub           *hub.SSEHub
	dataDir       string
	thumbCacheDir string
	log           *slog.Logger
}

// NewGenerator constructs a Generator.
func NewGenerator(clipRepo *database.PipelineClipRepo, h *hub.SSEHub, dataDir string) *Generator {
	return &Generator{
		clipRepo:      clipRepo,
		hub:           h,
		dataDir:       dataDir,
		thumbCacheDir: filepath.Join(dataDir, "thumb_cache"),
		log:           slog.With("component", "thumbnail.generator"),
	}
}

// resolveStrategy returns the appropriate Strategy for a pipeline mode.
func resolveStrategy(mode string) Strategy {
	switch mode {
	case "ai", "longform":
		return &LandscapeStrategy{}
	default:
		return &ShortsStrategy{}
	}
}

// GenerateBatch generates thumbnails for all clips in a run.
// Runs synchronously (called from a goroutine by the handler).
func (g *Generator) GenerateBatch(ctx context.Context, runID int64, clips []database.PipelineClip, cfg ThumbnailConfig, mode string, req *GenerateRequest) {
	jobKey := fmt.Sprintf("%d", runID)
	total := len(clips)
	fonts := DetectFonts()
	fontPath := ResolveFont(fonts, cfg.FontFamily)
	strategy := resolveStrategy(mode)

	successCount := 0
	errorCount := 0

	for i, clip := range clips {
		if ctx.Err() != nil {
			g.log.Warn("batch generation cancelled", "run_id", runID)
			return
		}

		// Emit "generating" status
		g.hub.Publish(jobKey, hub.SSEEvent{
			Type: "thumbnail_progress",
			Data: thumbnailProgressPayload{
				RunID:      runID,
				ClipID:     clip.ID,
				ClipIndex:  i + 1,
				TotalClips: total,
				Status:     "generating",
				Strategy:   strategy.Name(),
			},
		})

		thumbPath, err := g.generateOne(ctx, runID, clip, cfg, fontPath, strategy, i+1, total, req)
		if err != nil {
			errorCount++
			g.log.Error("thumbnail generation failed", "run_id", runID, "clip_id", clip.ID, "err", err)
			g.hub.Publish(jobKey, hub.SSEEvent{
				Type: "thumbnail_progress",
				Data: thumbnailProgressPayload{
					RunID:      runID,
					ClipID:     clip.ID,
					ClipIndex:  i + 1,
					TotalClips: total,
					Status:     "error",
					Strategy:   strategy.Name(),
					Error:      err.Error(),
				},
			})
			continue
		}

		successCount++
		// Update DB
		if err := g.clipRepo.UpdateThumbnailPath(ctx, clip.ID, thumbPath, strategy.Name()); err != nil {
			g.log.Error("failed to update thumbnail_path in DB", "clip_id", clip.ID, "err", err)
		}

		// Emit "done" status
		g.hub.Publish(jobKey, hub.SSEEvent{
			Type: "thumbnail_progress",
			Data: thumbnailProgressPayload{
				RunID:         runID,
				ClipID:        clip.ID,
				ClipIndex:     i + 1,
				TotalClips:    total,
				Status:        "done",
				Strategy:      strategy.Name(),
				ThumbnailPath: thumbPath,
			},
		})
	}

	// Emit batch_done
	g.hub.Publish(jobKey, hub.SSEEvent{
		Type: "thumbnail_progress",
		Data: thumbnailProgressPayload{
			RunID:        runID,
			ClipID:       0,
			ClipIndex:    total,
			TotalClips:   total,
			Status:       "batch_done",
			Strategy:     strategy.Name(),
			SuccessCount: successCount,
			ErrorCount:   errorCount,
		},
	})

	g.log.Info("batch thumbnail generation complete",
		"run_id", runID, "success", successCount, "errors", errorCount)
}

// GenerateSingle generates a thumbnail for a single clip. Returns the thumbnail path.
func (g *Generator) GenerateSingle(ctx context.Context, runID int64, clip database.PipelineClip, cfg ThumbnailConfig, mode string, req *GenerateRequest) (string, error) {
	fonts := DetectFonts()
	fontPath := ResolveFont(fonts, cfg.FontFamily)
	strategy := resolveStrategy(mode)

	thumbPath, err := g.generateOne(ctx, runID, clip, cfg, fontPath, strategy, 1, 1, req)
	if err != nil {
		return "", err
	}

	if err := g.clipRepo.UpdateThumbnailPath(ctx, clip.ID, thumbPath, strategy.Name()); err != nil {
		g.log.Error("failed to update thumbnail_path in DB", "clip_id", clip.ID, "err", err)
	}

	return thumbPath, nil
}

// generateOne generates a single thumbnail for one clip using the given strategy.
func (g *Generator) generateOne(ctx context.Context, runID int64, clip database.PipelineClip, cfg ThumbnailConfig, fontPath string, strategy Strategy, clipIndex int, totalClips int, req *GenerateRequest) (string, error) {
	// Source video path
	sourcePath := clip.FilePath
	if sourcePath == "" {
		return "", fmt.Errorf("clip %d has no file_path", clip.ID)
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source video not found: %s", sourcePath)
	}

	// Output path: same directory as clip file
	thumbDir := filepath.Dir(clip.FilePath)
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return "", fmt.Errorf("create thumbnail dir: %w", err)
	}
	// Derive thumbnail name from clip filename: foo.mp4 → foo_thumb.jpg
	base := strings.TrimSuffix(filepath.Base(clip.FilePath), filepath.Ext(clip.FilePath))
	outputPath := filepath.Join(thumbDir, base+"_thumb.jpg")

	// Seek to clip midpoint
	seekSec := clip.StartSec + (clip.DurationSec / 2)
	if seekSec <= 0 {
		seekSec = clip.StartSec
	}

	// Get text for this clip — landscape uses LandConfig.TextForClip, shorts uses cfg.TextForClip.
	var text string
	if strategy.Type() == StrategyLandscape && req != nil && req.LandscapeConfig != nil {
		text = req.LandscapeConfig.TextForClip(clip.ID, clipIndex)
	} else {
		text = cfg.TextForClip(clip.ID)
	}

	// For landscape strategy: resolve source to base image instead of video.
	if strategy.Type() == StrategyLandscape && req != nil && req.LandscapeConfig != nil {
		if req.LandscapeConfig.BaseImagePath != "" {
			if _, err := os.Stat(req.LandscapeConfig.BaseImagePath); err == nil {
				sourcePath = req.LandscapeConfig.BaseImagePath
			}
		}
		// If still pointing at the video, try downloading YouTube thumbnail.
		if sourcePath == clip.FilePath && req.ThumbnailURL != "" {
			downloaded, err := downloadThumbnail(req.ThumbnailURL, thumbDir)
			if err == nil {
				sourcePath = downloaded
			} else {
				g.log.Warn("failed to download YouTube thumbnail, using video frame", "err", err)
			}
		}
	}

	input := StrategyInput{
		SourcePath: sourcePath,
		OutputPath: outputPath,
		SeekSec:    seekSec,
		FontPath:   fontPath,
		Text:       text,
		Config:     cfg,
		ClipIndex:  clipIndex,
		TotalClips: totalClips,
	}
	if req != nil {
		input.LandConfig = req.LandscapeConfig
	}

	return strategy.Generate(ctx, input)
}

// downloadThumbnail downloads a remote image to dir and returns the local path.
func downloadThumbnail(rawURL, dir string) (string, error) {
	resp, err := http.Get(rawURL) //nolint:gosec // URL comes from server config, not user input
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	u, _ := url.Parse(rawURL)
	filename := path.Base(u.Path)
	if filename == "" || filename == "." {
		filename = "base_thumbnail.jpg"
	}

	destPath := filepath.Join(dir, "base_"+filename)
	f, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return destPath, nil
}
