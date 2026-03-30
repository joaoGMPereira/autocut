package processor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FFmpegProcessor wraps FFmpeg for common video operations.
// Kotlin ref: VideoProcessor + VerticalVideoProcessor (ProcessRunner.execute calls).
type FFmpegProcessor struct {
	binPath string
	exec    Executor
	log     *slog.Logger
}

// NewFFmpegProcessor creates an FFmpegProcessor.
// binPath defaults to "ffmpeg" when empty.
func NewFFmpegProcessor(binPath string) *FFmpegProcessor {
	if binPath == "" {
		binPath = "ffmpeg"
	}
	return &FFmpegProcessor{
		binPath: binPath,
		exec:    &DefaultExecutor{},
		log:     slog.With("component", "processor", "tool", "ffmpeg"),
	}
}

// newFFmpegProcessorWithExecutor is used in tests to inject a mock Executor.
func newFFmpegProcessorWithExecutor(binPath string, ex Executor) *FFmpegProcessor {
	if binPath == "" {
		binPath = "ffmpeg"
	}
	return &FFmpegProcessor{
		binPath: binPath,
		exec:    ex,
		log:     slog.With("component", "processor", "tool", "ffmpeg"),
	}
}

// formatDuration converts a time.Duration to an FFmpeg-compatible HH:MM:SS.mmm string.
// Kotlin ref: TimeUtils.formatTimestampWithMillis
func formatDuration(d time.Duration) string {
	total := d.Milliseconds()
	ms := total % 1000
	total /= 1000
	s := total % 60
	total /= 60
	m := total % 60
	h := total / 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// CutClip cuts a segment from input between start and end, writing to output.
// Uses stream copy (-c copy) so no re-encoding; very fast.
// Kotlin ref: VideoProcessor.cutClip (quality="copy" branch).
func (p *FFmpegProcessor) CutClip(input string, start, end time.Duration, output string) error {
	// Best-effort mkdir — ffmpeg itself will report a meaningful error on missing dir.
	_ = os.MkdirAll(filepath.Dir(output), 0o755)
	args := []string{
		"-ss", formatDuration(start),
		"-to", formatDuration(end),
		"-i", input,
		"-c", "copy",
		"-y",
		output,
	}
	p.log.Debug("CutClip", "op", "cut", "input", input, "start", start, "end", end)
	if _, err := p.exec.Run(p.binPath, args...); err != nil {
		p.log.Error("CutClip failed", "err", err, "input", input)
		return fmt.Errorf("cut clip: %w", err)
	}
	return nil
}

// ExtractFrame extracts a single frame at the given timestamp and saves it as an image.
// Kotlin ref: VideoProcessor.extractFrame.
func (p *FFmpegProcessor) ExtractFrame(input string, at time.Duration, output string) error {
	_ = os.MkdirAll(filepath.Dir(output), 0o755)
	args := []string{
		"-ss", formatDuration(at),
		"-i", input,
		"-frames:v", "1",
		"-q:v", "2",
		"-y",
		output,
	}
	p.log.Debug("ExtractFrame", "op", "frame", "input", input, "at", at)
	if _, err := p.exec.Run(p.binPath, args...); err != nil {
		p.log.Error("ExtractFrame failed", "err", err, "input", input)
		return fmt.Errorf("extract frame: %w", err)
	}
	return nil
}

// MergeClips concatenates multiple input files into a single output using the
// concat demuxer (stream copy — no re-encoding).
// Kotlin ref: VideoProcessor.concatenateClips.
func (p *FFmpegProcessor) MergeClips(inputs []string, output string) error {
	if len(inputs) == 0 {
		return fmt.Errorf("merge clips: no inputs provided")
	}
	_ = os.MkdirAll(filepath.Dir(output), 0o755)

	// Write temp concat list file
	listFile := filepath.Join(os.TempDir(), fmt.Sprintf("autocut_concat_%d.txt", time.Now().UnixNano()))
	var sb strings.Builder
	for _, in := range inputs {
		abs, err := filepath.Abs(in)
		if err != nil {
			return fmt.Errorf("abs path for %q: %w", in, err)
		}
		sb.WriteString(fmt.Sprintf("file '%s'\n", abs))
	}
	if err := os.WriteFile(listFile, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("write concat list: %w", err)
	}
	defer os.Remove(listFile)

	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y",
		output,
	}
	p.log.Debug("MergeClips", "op", "merge", "count", len(inputs))
	if _, err := p.exec.Run(p.binPath, args...); err != nil {
		p.log.Error("MergeClips failed", "err", err)
		return fmt.Errorf("merge clips: %w", err)
	}
	return nil
}

// ApplySpeedZones applies per-zone speed changes via FFmpeg filter_complex.
// Each zone uses setpts (video) + atempo chain (audio).
// Kotlin ref: VerticalVideoProcessor.speedUp + SpeedChainBuilder.buildAtempoChain.
func (p *FFmpegProcessor) ApplySpeedZones(input string, zones []SpeedZone, output string) error {
	if len(zones) == 0 {
		return fmt.Errorf("apply speed zones: no zones provided")
	}
	_ = os.MkdirAll(filepath.Dir(output), 0o755)

	// For simplicity we apply a single uniform speed across zones by building
	// a filter_complex that trims + speeds each zone and concatenates them.
	// Full multi-zone implementation builds one trim+setpts+atempo per zone.
	var filterParts []string
	var vLabels, aLabels []string

	for i, z := range zones {
		startSec := z.Start.Seconds()
		endSec := z.End.Seconds()
		speed := z.Speed

		vTrim := fmt.Sprintf("[0:v]trim=start=%.3f:end=%.3f,setpts=PTS/%.4f[v%d]", startSec, endSec, speed, i)
		aTrim := fmt.Sprintf("[0:a]atrim=start=%.3f:end=%.3f,%s,asetpts=PTS-STARTPTS[a%d]",
			startSec, endSec, buildAtempoChain(speed), i)

		filterParts = append(filterParts, vTrim, aTrim)
		vLabels = append(vLabels, fmt.Sprintf("[v%d]", i))
		aLabels = append(aLabels, fmt.Sprintf("[a%d]", i))
	}

	n := len(zones)
	concatFilter := fmt.Sprintf("%s%sconcat=n=%d:v=1:a=1[outv][outa]",
		strings.Join(vLabels, ""), strings.Join(aLabels, ""), n)
	filterParts = append(filterParts, concatFilter)

	filterComplex := strings.Join(filterParts, ";")

	args := []string{
		"-i", input,
		"-filter_complex", filterComplex,
		"-map", "[outv]",
		"-map", "[outa]",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-y",
		output,
	}
	p.log.Debug("ApplySpeedZones", "op", "speed", "zones", len(zones))
	if _, err := p.exec.Run(p.binPath, args...); err != nil {
		p.log.Error("ApplySpeedZones failed", "err", err)
		return fmt.Errorf("apply speed zones: %w", err)
	}
	return nil
}

// RemoveSilence detects and removes silent sections using silencedetect filter.
// Kotlin ref: AudioSilenceDetector.detectSilences.
//
// threshold is in dB (e.g., -30), minDuration is the minimum silence length to cut.
func (p *FFmpegProcessor) RemoveSilence(input string, threshold float64, minDuration time.Duration, output string) error {
	_ = os.MkdirAll(filepath.Dir(output), 0o755)

	// Step 1: Run silencedetect to collect timestamps.
	detectArgs := []string{
		"-i", input,
		"-af", fmt.Sprintf("silencedetect=noise=%.1fdB:d=%.3f", threshold, minDuration.Seconds()),
		"-f", "null",
		"-",
	}
	p.log.Debug("RemoveSilence detect", "op", "silence_detect", "threshold", threshold, "minDuration", minDuration)
	out, err := p.exec.Run(p.binPath, detectArgs...)
	if err != nil {
		// silencedetect exits non-zero when it prints to stderr; tolerate it if output is present.
		if len(out) == 0 {
			return fmt.Errorf("silence detect: %w", err)
		}
	}

	// Step 2: Parse silence_start / silence_end timestamps from combined output.
	type interval struct{ start, end float64 }
	silences := parseSilenceIntervals(string(out))

	if len(silences) == 0 {
		p.log.Info("RemoveSilence: no silences found, copying input as-is")
		// No silences — copy file directly.
		cpArgs := []string{"-i", input, "-c", "copy", "-y", output}
		if _, err := p.exec.Run(p.binPath, cpArgs...); err != nil {
			return fmt.Errorf("copy (no silences): %w", err)
		}
		return nil
	}

	// Step 3: Build keep-segments and merge via concat demuxer.
	keeps := buildKeepSegments(silences)
	if len(keeps) == 0 {
		return fmt.Errorf("remove silence: all content is silent")
	}

	// Write concat list with trim clips written to temp files.
	tmpDir := os.TempDir()
	var listEntries []string
	var tempFiles []string

	for i, seg := range keeps {
		tmpOut := filepath.Join(tmpDir, fmt.Sprintf("autocut_seg_%d_%d.mp4", time.Now().UnixNano(), i))
		tempFiles = append(tempFiles, tmpOut)

		segArgs := []string{
			"-ss", fmt.Sprintf("%.3f", seg.start),
			"-to", fmt.Sprintf("%.3f", seg.end),
			"-i", input,
			"-c", "copy",
			"-y", tmpOut,
		}
		if _, err := p.exec.Run(p.binPath, segArgs...); err != nil {
			p.log.Error("RemoveSilence segment cut failed", "err", err, "seg", i)
			return fmt.Errorf("remove silence segment %d: %w", i, err)
		}
		abs, _ := filepath.Abs(tmpOut)
		listEntries = append(listEntries, fmt.Sprintf("file '%s'", abs))
	}
	defer func() {
		for _, f := range tempFiles {
			os.Remove(f)
		}
	}()

	listFile := filepath.Join(tmpDir, fmt.Sprintf("autocut_silence_list_%d.txt", time.Now().UnixNano()))
	if err := os.WriteFile(listFile, []byte(strings.Join(listEntries, "\n")), 0o644); err != nil {
		return fmt.Errorf("write silence concat list: %w", err)
	}
	defer os.Remove(listFile)

	mergeArgs := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y", output,
	}
	p.log.Debug("RemoveSilence merge", "op", "silence_merge", "segments", len(keeps))
	if _, err := p.exec.Run(p.binPath, mergeArgs...); err != nil {
		p.log.Error("RemoveSilence merge failed", "err", err)
		return fmt.Errorf("remove silence merge: %w", err)
	}
	return nil
}

// ---- internal helpers ----

type silenceInterval struct{ start, end float64 }

// parseSilenceIntervals extracts silence_start/silence_end pairs from FFmpeg stderr.
// Kotlin ref: AudioSilenceDetector regex parsing.
func parseSilenceIntervals(output string) []silenceInterval {
	var starts, ends []float64
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.Index(line, "silence_start:"); idx >= 0 {
			var v float64
			if _, err := fmt.Sscanf(strings.TrimSpace(line[idx+14:]), "%f", &v); err == nil {
				starts = append(starts, v)
			}
		}
		if idx := strings.Index(line, "silence_end:"); idx >= 0 {
			var v float64
			if _, err := fmt.Sscanf(strings.TrimSpace(line[idx+12:]), "%f", &v); err == nil {
				ends = append(ends, v)
			}
		}
	}
	var result []silenceInterval
	for i := range starts {
		end := starts[i] + 1.0 // fallback
		if i < len(ends) {
			end = ends[i]
		}
		result = append(result, silenceInterval{start: starts[i], end: end})
	}
	return result
}

// buildKeepSegments inverts silence intervals into keep segments.
// Kotlin ref: AudioSilenceDetector.buildSegments (KEEP entries).
func buildKeepSegments(silences []silenceInterval) []silenceInterval {
	var keeps []silenceInterval
	cursor := 0.0
	for _, s := range silences {
		if s.start > cursor+0.05 {
			keeps = append(keeps, silenceInterval{start: cursor, end: s.start})
		}
		cursor = s.end
	}
	// Trailing keep — use a very large end so ffmpeg reads until EOF.
	keeps = append(keeps, silenceInterval{start: cursor, end: 1e9})
	return keeps
}

// buildAtempoChain produces a chained atempo filter string for speeds outside 0.5-2.0.
// Kotlin ref: SpeedChainBuilder.buildAtempoChain.
func buildAtempoChain(speed float64) string {
	if speed <= 2.0 && speed >= 0.5 {
		return fmt.Sprintf("atempo=%.4f", speed)
	}
	var parts []string
	remaining := speed
	for remaining > 2.0 {
		parts = append(parts, "atempo=2.0000")
		remaining /= 2.0
	}
	if remaining > 1.001 {
		parts = append(parts, fmt.Sprintf("atempo=%.4f", remaining))
	}
	return strings.Join(parts, ",")
}
