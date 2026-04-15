package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ClipRequest contains everything needed to generate a single clip with effects.
type ClipRequest struct {
	RunID       int64
	ClipID      int64
	VideoPath   string
	StartSec    float64         // from highlight adj_start_sec
	EndSec      float64         // from highlight adj_end_sec
	ChannelCfg  database.ChannelConfig
	ModeCfg     json.RawMessage // raw ModeConfig JSON
	DataDir     string          // root dir for output files ({dataDir}/clips/)
	OnProgress  func(float64)   // callback: percent 0–100
}

// GenerateClip cuts a single clip segment and applies all configured effect layers.
// Unlike preview, clips are not cached — each clip is unique per highlight.
// Returns the path to the output file on success.
func GenerateClip(ctx context.Context, req ClipRequest) (string, error) {
	log := slog.With("component", "clip", "run_id", req.RunID, "clip_id", req.ClipID)

	clipDir := filepath.Join(req.DataDir, "clips")
	if err := os.MkdirAll(clipDir, 0o755); err != nil {
		return "", fmt.Errorf("create clips dir: %w", err)
	}

	outPath := filepath.Join(clipDir, fmt.Sprintf("%d_%d.mp4", req.RunID, req.ClipID))

	startSec := int(req.StartSec)
	durationSec := int64(req.EndSec - req.StartSec)
	if durationSec <= 0 {
		return "", fmt.Errorf("invalid clip duration: start=%.1f end=%.1f", req.StartSec, req.EndSec)
	}

	// Build ffmpeg args using the shared effect chain
	effectReq := EffectRequest{
		VideoPath:   req.VideoPath,
		StartSec:    startSec,
		DurationSec: durationSec,
		ChannelCfg:  req.ChannelCfg,
		OutputPath:  outPath,
	}
	args, cleanups := BuildEffectChain(log, effectReq)
	for _, fn := range cleanups {
		defer fn()
	}

	log.Info("generating clip", "output", outPath, "start_sec", startSec, "duration", durationSec)

	err := RunFFmpeg(ctx, args, float64(durationSec), req.OnProgress)
	if err != nil {
		_ = os.Remove(outPath)
		return "", fmt.Errorf("ffmpeg clip: %w", err)
	}

	log.Info("clip generated", "path", outPath)
	return outPath, nil
}
