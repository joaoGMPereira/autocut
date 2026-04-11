package ai

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// SceneChange represents a detected scene transition at a specific timestamp.
type SceneChange struct {
	TimeSec float64 `json:"time_sec"`
	Score   float64 `json:"score"` // 0.0–1.0 as reported by FFmpeg
}

// VisualDetector detects scene changes in a video file using FFmpeg's scene
// detection filter.
type VisualDetector struct {
	FFmpegPath string
	Threshold  float64 // scene change sensitivity; default 0.4
	logger     *slog.Logger
}

// NewVisualDetector creates a VisualDetector using the given ffmpeg binary path.
// Threshold defaults to 0.4 when zero.
func NewVisualDetector(ffmpegPath string) *VisualDetector {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &VisualDetector{
		FFmpegPath: ffmpegPath,
		Threshold:  0.4,
		logger:     slog.With("component", "visual_detector"),
	}
}

// reSceneScore matches "lavfi.scene_score=0.453821" in FFmpeg metadata output.
var reSceneScore = regexp.MustCompile(`lavfi\.scene_score=([\d.]+)`)

// rePtsTime matches "pts_time:12.345" in FFmpeg metadata output.
var rePtsTime = regexp.MustCompile(`pts_time:([\d.]+)`)

// DetectSceneChanges runs FFmpeg's select+metadata filter to find frames where
// the scene changes significantly (score > Threshold). Returns SceneChange
// entries sorted by TimeSec ascending.
//
// FFmpeg command used:
//
//	ffmpeg -i {videoPath} -vf "select='gt(scene,{threshold})',metadata=print:file=-" -f null -
//
// The context is respected via exec.CommandContext.
func (d *VisualDetector) DetectSceneChanges(ctx context.Context, videoPath string) ([]SceneChange, error) {
	d.logger.Info("visual detect started", "video", videoPath, "threshold", d.Threshold)

	threshold := d.Threshold
	if threshold <= 0 {
		threshold = 0.4
	}

	// Build the filter expression. The backslash-comma is required by FFmpeg
	// when embedding a select expression inside a filtergraph argument.
	filterExpr := fmt.Sprintf("select='gt(scene\\,%g)',metadata=print:file=-", threshold)
	args := []string{
		"-i", videoPath,
		"-vf", filterExpr,
		"-f", "null",
		"-",
	}

	out, err := d.runFFmpeg(ctx, args)
	if err != nil {
		// FFmpeg exits non-zero on null output — only treat as error if no
		// scene metadata was produced at all.
		if !bytes.Contains(out, []byte("pts_time")) && !bytes.Contains(out, []byte("scene_score")) {
			d.logger.Error("visual detect ffmpeg failed", "err", err)
			return nil, fmt.Errorf("visual_detector: ffmpeg: %w", err)
		}
	}

	changes := d.parseMetadataOutput(out)
	d.logger.Info("visual detect done", "scene_changes", len(changes))
	return changes, nil
}

// parseMetadataOutput scans FFmpeg metadata=print output for pts_time and
// scene_score pairs. Each detected frame produces one SceneChange entry.
//
// Example output block:
//
//	frame:42   pts:1050      pts_time:42.0
//	lavfi.scene_score=0.512345
func (d *VisualDetector) parseMetadataOutput(out []byte) []SceneChange {
	var (
		changes  []SceneChange
		pendingT float64
		hasTime  bool
	)

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if m := rePtsTime.FindStringSubmatch(line); len(m) == 2 {
			t, err := strconv.ParseFloat(m[1], 64)
			if err == nil {
				pendingT = t
				hasTime = true
			}
		}

		if m := reSceneScore.FindStringSubmatch(line); len(m) == 2 {
			score, err := strconv.ParseFloat(m[1], 64)
			if err == nil && hasTime {
				changes = append(changes, SceneChange{
					TimeSec: pendingT,
					Score:   clampScore(score),
				})
				hasTime = false
			}
		}
	}

	return changes
}

// runFFmpeg executes ffmpeg with the given args and returns combined output.
func (d *VisualDetector) runFFmpeg(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.FFmpegPath, args...)
	return cmd.CombinedOutput()
}
