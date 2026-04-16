package processor

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// PreviewRequest contains everything needed to generate a preview video.
type PreviewRequest struct {
	RunID          int64
	VideoPath      string
	DurationSec    int64
	ChannelCfg     database.ChannelConfig
	ModeCfg        json.RawMessage // raw ModeConfig JSON from POST body
	DataDir        string          // root dir for output files ({dataDir}/previews/)
	OnProgress     func(float64)   // callback: percent 0–100
	BlurEdgePct    float64         // 0=disabled, 1-100=edge blur percentage
	NoiseStrength  float64         // 0=disabled, 1-10=noise strength
	TranscriptPath string          // optional: path to transcript JSON for real captions
}

// GeneratePreview produces a preview MP4 for a pipeline run.
// It selects a representative 2-minute segment starting at ~20% of the video,
// applies all configured effect layers, and caches the result by config hash.
// Returns the path to the output file on success.
func GeneratePreview(ctx context.Context, req PreviewRequest) (string, error) {
	log := slog.With("component", "preview", "run_id", req.RunID)

	hash, err := computeCacheHash(req.ChannelCfg, req.ModeCfg, req.TranscriptPath)
	if err != nil {
		return "", fmt.Errorf("compute cache hash: %w", err)
	}

	previewDir := filepath.Join(req.DataDir, "previews")
	if err := os.MkdirAll(previewDir, 0o755); err != nil {
		return "", fmt.Errorf("create previews dir: %w", err)
	}

	outPath := filepath.Join(previewDir, fmt.Sprintf("%d_%s.mp4", req.RunID, hash))

	// Cache hit
	if _, err := os.Stat(outPath); err == nil {
		log.Info("preview cache hit", "path", outPath)
		return outPath, nil
	}

	// Smart start position: ~20% into the video to skip intros/credits
	startSec := max(10, int(float64(req.DurationSec)*0.20))
	if int64(startSec) >= req.DurationSec {
		startSec = 0 // fallback for very short videos
	}
	segmentDuration := min(int64(15), req.DurationSec-int64(startSec))
	if segmentDuration <= 0 {
		segmentDuration = req.DurationSec
		startSec = 0
	}

	// Resolve transcript for caption burn-in (on-demand whisper if needed)
	transcriptPath, transcriptCleanup, tErr := ResolveTranscript(
		req.VideoPath, startSec, segmentDuration,
		req.DataDir, req.TranscriptPath, req.ChannelCfg.PreviewCaptionsEnabled,
	)
	if tErr != nil {
		log.Warn("transcript resolution failed, captions will use placeholder", "err", tErr)
	}
	if transcriptCleanup != nil {
		defer transcriptCleanup()
	}

	// Build ffmpeg args using the shared effect chain
	effectReq := EffectRequest{
		VideoPath:      req.VideoPath,
		StartSec:       startSec,
		DurationSec:    segmentDuration,
		ChannelCfg:     req.ChannelCfg,
		OutputPath:     outPath,
		BlurEdgePct:    req.BlurEdgePct,
		NoiseStrength:  req.NoiseStrength,
		TranscriptPath: transcriptPath,
	}
	args, cleanups := BuildEffectChain(log, effectReq)
	for _, fn := range cleanups {
		defer fn()
	}

	log.Info("generating preview", "output", outPath, "start_sec", startSec, "duration", segmentDuration)

	err = RunFFmpeg(ctx, args, float64(segmentDuration), req.OnProgress)
	if err != nil {
		_ = os.Remove(outPath)
		return "", fmt.Errorf("ffmpeg preview: %w", err)
	}

	log.Info("preview generated", "path", outPath)
	return outPath, nil
}

// computeCacheHash returns MD5 hex of ChannelConfig + ModeConfig + transcript path.
// Including transcript path ensures the cache busts when a transcript becomes available.
func computeCacheHash(cc database.ChannelConfig, modeCfg json.RawMessage, transcriptPath string) (string, error) {
	ccJSON, err := json.Marshal(cc)
	if err != nil {
		return "", err
	}
	combined := append(ccJSON, modeCfg...)
	combined = append(combined, []byte(transcriptPath)...)
	return fmt.Sprintf("%x", md5.Sum(combined)), nil
}
