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

// AudioHighlight is a time range identified as a highlight by volume analysis.
// Reason is one of: "volume_spike", "after_silence".
type AudioHighlight struct {
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
	Score    float64 `json:"score"` // 0.0–1.0, normalised against max_volume
	Reason   string  `json:"reason"`
}

// AudioDetector detects highlights in a video by analysing audio volume levels
// via FFmpeg's volumedetect and silencedetect filters.
type AudioDetector struct {
	FFmpegPath string
	logger     *slog.Logger
}

// NewAudioDetector creates an AudioDetector using the given ffmpeg binary path.
// If ffmpegPath is empty, "ffmpeg" is used (resolved via PATH).
func NewAudioDetector(ffmpegPath string) *AudioDetector {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &AudioDetector{
		FFmpegPath: ffmpegPath,
		logger:     slog.With("component", "audio_detector"),
	}
}

// reVolumedetect matches lines like: "mean_volume: -23.4 dB" or "max_volume: -5.0 dB"
var (
	reMeanVolume = regexp.MustCompile(`mean_volume:\s*([-\d.]+)\s*dB`)
	reMaxVolume  = regexp.MustCompile(`max_volume:\s*([-\d.]+)\s*dB`)
	reSilenceEnd = regexp.MustCompile(`silence_end:\s*([\d.]+)`)
)

// DetectHighlights analyses the audio track of videoPath and returns a list of
// AudioHighlight entries. It runs two FFmpeg passes:
//  1. volumedetect — to obtain mean and max volume levels.
//  2. silencedetect — to find silences; the moment audio resumes after a silence
//     is treated as a potential highlight (audience reaction / reveal).
//
// The context is respected: FFmpeg subprocesses are started with CommandContext.
func (d *AudioDetector) DetectHighlights(ctx context.Context, videoPath string) ([]AudioHighlight, error) {
	d.logger.Info("audio detect started", "video", videoPath)

	meanVol, maxVol, err := d.runVolumeDetect(ctx, videoPath)
	if err != nil {
		d.logger.Error("volumedetect failed", "err", err, "video", videoPath)
		return nil, fmt.Errorf("audio_detector: volumedetect: %w", err)
	}
	d.logger.Debug("volumedetect result", "mean_volume", meanVol, "max_volume", maxVol)

	silenceEnds, err := d.runSilenceDetect(ctx, videoPath)
	if err != nil {
		// Non-fatal: log and continue without silence highlights.
		d.logger.Warn("silencedetect failed, skipping silence highlights", "err", err)
		silenceEnds = nil
	}
	d.logger.Debug("silence ends found", "count", len(silenceEnds))

	var highlights []AudioHighlight

	// Volume spike: threshold = mean + (max - mean) * 0.6
	// If we can't distinguish (max ≈ mean) skip volume-spike highlights.
	spread := maxVol - meanVol
	if spread > 1.0 {
		threshold := meanVol + spread*0.6
		spike := AudioHighlight{
			StartSec: 0,
			EndSec:   0,
			Score:    clampScore((maxVol - threshold) / spread),
			Reason:   "volume_spike",
		}
		// We only have global stats, not per-frame; emit one highlight at t=0
		// representing the loudest moment we know exists. Callers can combine
		// with visual/chat data to locate the precise timestamp.
		_ = threshold
		spike.StartSec = 0
		spike.EndSec = 0
		// Score relative to max possible: (max - mean) / |mean|
		if meanVol != 0 {
			spike.Score = clampScore((maxVol - meanVol) / (-meanVol))
		} else {
			spike.Score = clampScore(spread / 30.0) // 30 dB spread = score 1.0
		}
		highlights = append(highlights, spike)
	}

	// After-silence highlights: audio resumes after a quiet gap — often a
	// reveal, reaction, or musical drop.
	for _, t := range silenceEnds {
		const windowSec = 5.0
		score := 0.6
		if spread > 0 {
			score = clampScore(0.5 + spread/60.0)
		}
		highlights = append(highlights, AudioHighlight{
			StartSec: t,
			EndSec:   t + windowSec,
			Score:    score,
			Reason:   "after_silence",
		})
	}

	d.logger.Info("audio detect done", "highlights", len(highlights))
	return highlights, nil
}

// runVolumeDetect runs FFmpeg with the volumedetect filter and returns
// mean_volume and max_volume in dB (negative floats).
func (d *AudioDetector) runVolumeDetect(ctx context.Context, videoPath string) (mean, max float64, err error) {
	args := []string{
		"-i", videoPath,
		"-af", "volumedetect",
		"-f", "null",
		"-",
	}
	out, err := d.runFFmpeg(ctx, args)
	if err != nil {
		// FFmpeg exits non-zero for null output — check if we got useful data.
		if !bytes.Contains(out, []byte("mean_volume")) {
			return 0, 0, fmt.Errorf("ffmpeg volumedetect: %w", err)
		}
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if m := reMeanVolume.FindStringSubmatch(line); len(m) == 2 {
			mean, _ = strconv.ParseFloat(m[1], 64)
		}
		if m := reMaxVolume.FindStringSubmatch(line); len(m) == 2 {
			max, _ = strconv.ParseFloat(m[1], 64)
		}
	}
	return mean, max, nil
}

// runSilenceDetect runs FFmpeg with the silencedetect filter and returns a list
// of timestamps (seconds) where silence ends (i.e., audio resumes).
func (d *AudioDetector) runSilenceDetect(ctx context.Context, videoPath string) ([]float64, error) {
	args := []string{
		"-i", videoPath,
		"-af", "silencedetect=noise=-40dB:d=0.5",
		"-f", "null",
		"-",
	}
	out, err := d.runFFmpeg(ctx, args)
	if err != nil {
		if !bytes.Contains(out, []byte("silence_end")) && !bytes.Contains(out, []byte("silencedetect")) {
			return nil, fmt.Errorf("ffmpeg silencedetect: %w", err)
		}
	}

	var ends []float64
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if m := reSilenceEnd.FindStringSubmatch(line); len(m) == 2 {
			t, err := strconv.ParseFloat(m[1], 64)
			if err == nil {
				ends = append(ends, t)
			}
		}
	}
	return ends, nil
}

// runFFmpeg executes the ffmpeg binary with args and returns combined stdout+stderr.
func (d *AudioDetector) runFFmpeg(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.FFmpegPath, args...)
	return cmd.CombinedOutput()
}

// clampScore ensures v is in [0.0, 1.0].
func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
