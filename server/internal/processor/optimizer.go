package processor

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

// OptimizerConfig configures video optimization behaviour.
// Kotlin ref: OptimizationConfig (silence + speed fields).
type OptimizerConfig struct {
	// SpeedZones lists time ranges with custom playback speeds.
	SpeedZones []SpeedZone
	// SilenceThreshold is the dB level below which audio is considered silent.
	// Pass 0 to skip silence removal.
	SilenceThreshold float64
	// MinSilenceDuration is the minimum length of a silence region to remove.
	MinSilenceDuration time.Duration
}

// ffmpegInterface is an internal interface that allows tests to mock FFmpegProcessor.
type ffmpegInterface interface {
	CutClip(input string, start, end time.Duration, output string) error
	RemoveSilence(input string, threshold float64, minDuration time.Duration, output string) error
	ApplySpeedZones(input string, zones []SpeedZone, output string) error
}

// VideoOptimizerProcessor applies silence removal and speed zones to a video.
// Kotlin ref: VideoOptimizerProcessor.optimize.
type VideoOptimizerProcessor struct {
	ffmpeg     ffmpegInterface
	ffprobeBin string
	log        *slog.Logger
}

// NewVideoOptimizerProcessor creates a VideoOptimizerProcessor.
// ffmpegPath defaults to "ffmpeg" when empty.
func NewVideoOptimizerProcessor(ffmpegPath string) *VideoOptimizerProcessor {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &VideoOptimizerProcessor{
		ffmpeg:     NewFFmpegProcessor(ffmpegPath),
		ffprobeBin: "ffprobe",
		log:        slog.With("component", "processor", "tool", "optimizer"),
	}
}

// newVideoOptimizerProcessorWithMock injects a mock for tests.
func newVideoOptimizerProcessorWithMock(f ffmpegInterface, ffprobeBin string) *VideoOptimizerProcessor {
	return &VideoOptimizerProcessor{
		ffmpeg:     f,
		ffprobeBin: ffprobeBin,
		log:        slog.With("component", "processor", "tool", "optimizer"),
	}
}

// Optimize runs silence removal (when SilenceThreshold != 0) and speed zones (when non-empty),
// writing the result to output. Returns an OptimizeResult with duration statistics.
//
// Duration probing is best-effort: failures are logged but do not abort processing.
// Kotlin ref: VideoOptimizerProcessor.optimize pipeline.
func (p *VideoOptimizerProcessor) Optimize(input string, cfg OptimizerConfig, output string) (*OptimizeResult, error) {
	// Probe original duration — best effort, 0 on failure.
	origDuration, probeErr := p.probeDuration(input)
	if probeErr != nil {
		p.log.Warn("Optimize: probe original duration failed (continuing)", "err", probeErr, "input", input)
	}

	current := input
	removedSilences := 0

	// Stage 1: Silence removal.
	if cfg.SilenceThreshold != 0 {
		minDur := cfg.MinSilenceDuration
		if minDur == 0 {
			minDur = 300 * time.Millisecond
		}

		silenceOut := output + ".silence_pass.mp4"
		p.log.Info("Optimize: removing silence", "threshold", cfg.SilenceThreshold, "minDuration", minDur)
		if err := p.ffmpeg.RemoveSilence(current, cfg.SilenceThreshold, minDur, silenceOut); err != nil {
			p.log.Error("Optimize: RemoveSilence failed", "err", err)
			return nil, fmt.Errorf("optimize silence removal: %w", err)
		}
		defer os.Remove(silenceOut)

		current = silenceOut

		// Approximate removed silences via duration difference.
		if origDuration > 0 {
			silenceDur, pErr := p.probeDuration(silenceOut)
			if pErr == nil {
				diff := origDuration - silenceDur
				if diff > 0 {
					removedSilences = int(diff.Seconds())
				}
			}
		}
	}

	// Stage 2: Speed zones.
	if len(cfg.SpeedZones) > 0 {
		p.log.Info("Optimize: applying speed zones", "zones", len(cfg.SpeedZones))
		if err := p.ffmpeg.ApplySpeedZones(current, cfg.SpeedZones, output); err != nil {
			p.log.Error("Optimize: ApplySpeedZones failed", "err", err)
			return nil, fmt.Errorf("optimize speed zones: %w", err)
		}
	} else if current != input {
		// Only silence removal ran — move intermediate file to final output.
		if err := osCopy(current, output); err != nil {
			return nil, fmt.Errorf("optimize finalize: %w", err)
		}
	} else {
		// No operations — passthrough copy.
		p.log.Info("Optimize: no operations configured, copying input")
		if err := osCopy(input, output); err != nil {
			return nil, fmt.Errorf("optimize passthrough: %w", err)
		}
	}

	// Probe final duration — best effort.
	finalDuration, err := p.probeDuration(output)
	if err != nil {
		p.log.Warn("Optimize: probe final duration failed", "err", err)
		finalDuration = origDuration
	}

	return &OptimizeResult{
		OriginalDuration: origDuration,
		FinalDuration:    finalDuration,
		RemovedSilences:  removedSilences,
	}, nil
}

// probeDuration uses ffprobe to read the duration of a media file.
// Kotlin ref: VideoUtils.getVideoDurationOrDefault via ffprobe JSON output.
func (p *VideoOptimizerProcessor) probeDuration(path string) (time.Duration, error) {
	out, err := exec.Command(p.ffprobeBin,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		path,
	).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe %q: %w\noutput: %s", path, err, string(out))
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return 0, fmt.Errorf("ffprobe parse: %w", err)
	}

	var secs float64
	if _, err := fmt.Sscanf(result.Format.Duration, "%f", &secs); err != nil {
		return 0, fmt.Errorf("ffprobe duration parse %q: %w", result.Format.Duration, err)
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// osCopy performs a raw file copy using os primitives.
func osCopy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src %q: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst %q: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %q → %q: %w", src, dst, err)
	}
	return nil
}
