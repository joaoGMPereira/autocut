package effects

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// EffectsService applies video effects using FFmpeg.
// All methods are stateless: they accept input path(s), apply an FFmpeg filter,
// and write the result to outputPath.
// Kotlin refs: VideoUniquenessProcessor, VideoOptimizerProcessor,
//
//	TextOverlayProcessor, VideoOverlayProcessor.
type EffectsService struct {
	binPath string
	exec    Executor
	log     *slog.Logger
}

// NewEffectsService creates an EffectsService.
// ffmpegBinPath defaults to "ffmpeg" when empty.
func NewEffectsService(ffmpegBinPath string) *EffectsService {
	if ffmpegBinPath == "" {
		ffmpegBinPath = "ffmpeg"
	}
	return &EffectsService{
		binPath: ffmpegBinPath,
		exec:    &DefaultExecutor{},
		log:     slog.With("component", "effects"),
	}
}

// newEffectsServiceWithExecutor injects a mock executor for testing.
func newEffectsServiceWithExecutor(binPath string, ex Executor) *EffectsService {
	if binPath == "" {
		binPath = "ffmpeg"
	}
	return &EffectsService{binPath: binPath, exec: ex, log: slog.With("component", "effects")}
}

// ── AntiDup ──────────────────────────────────────────────────────────────────

// AntiDup burns a near-invisible caption into the video to defeat content-ID
// fingerprinting by making each render unique.
// Kotlin ref: VideoUniquenessProcessor.
func (s *EffectsService) AntiDup(videoPath, captionText, outputPath string) error {
	_ = os.MkdirAll(filepath.Dir(outputPath), 0o755)

	// Escape single quotes for the drawtext filter value.
	escaped := strings.ReplaceAll(captionText, `'`, `\'`)
	drawtext := fmt.Sprintf(
		"drawtext=text='%s':fontsize=8:fontcolor=white@0.01:x=10:y=10",
		escaped,
	)

	args := []string{
		"-loglevel", "error",
		"-i", videoPath,
		"-vf", drawtext,
		"-c:a", "copy",
		"-y", outputPath,
	}
	s.log.Debug("AntiDup", "op", "anti_dup", "input", videoPath, "output", outputPath)
	if _, err := s.exec.Run(s.binPath, args...); err != nil {
		s.log.Error("AntiDup failed", "err", err, "input", videoPath)
		return fmt.Errorf("anti-dup: %w", err)
	}
	return nil
}

// ── SpeedZones ────────────────────────────────────────────────────────────────

// SpeedZones applies per-segment variable speed changes.
// Each segment is trimmed and accelerated/decelerated independently, then
// the results are concatenated.
// Kotlin ref: VideoOptimizerProcessor speed-zone logic.
func (s *EffectsService) SpeedZones(videoPath string, segments []SpeedSegment, outputPath string) error {
	if len(segments) == 0 {
		return fmt.Errorf("speed-zones: no segments provided")
	}
	_ = os.MkdirAll(filepath.Dir(outputPath), 0o755)

	var filterParts []string
	var vLabels, aLabels []string

	for i, seg := range segments {
		speed := seg.Speed
		if speed <= 0 {
			speed = 1.0
		}
		vf := fmt.Sprintf(
			"[0:v]trim=start=%.3f:end=%.3f,setpts=PTS/%.4f,setpts=PTS-STARTPTS[v%d]",
			seg.StartSec, seg.EndSec, speed, i,
		)
		af := fmt.Sprintf(
			"[0:a]atrim=start=%.3f:end=%.3f,%s,asetpts=PTS-STARTPTS[a%d]",
			seg.StartSec, seg.EndSec, buildAtempoChain(speed), i,
		)
		filterParts = append(filterParts, vf, af)
		vLabels = append(vLabels, fmt.Sprintf("[v%d]", i))
		aLabels = append(aLabels, fmt.Sprintf("[a%d]", i))
	}

	n := len(segments)
	concat := fmt.Sprintf(
		"%s%sconcat=n=%d:v=1:a=1[outv][outa]",
		strings.Join(vLabels, ""), strings.Join(aLabels, ""), n,
	)
	filterParts = append(filterParts, concat)

	args := []string{
		"-i", videoPath,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "[outv]",
		"-map", "[outa]",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-y", outputPath,
	}
	s.log.Debug("SpeedZones", "op", "speed_zones", "segments", n, "input", videoPath)
	if _, err := s.exec.Run(s.binPath, args...); err != nil {
		s.log.Error("SpeedZones failed", "err", err, "input", videoPath)
		return fmt.Errorf("speed-zones: %w", err)
	}
	return nil
}

// buildAtempoChain produces a chained atempo filter for speeds outside 0.5-2.0.
// atempo only accepts values in [0.5, 2.0], so extreme speeds require chaining.
func buildAtempoChain(speed float64) string {
	if speed >= 0.5 && speed <= 2.0 {
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
	if len(parts) == 0 {
		return "atempo=1.0000"
	}
	return strings.Join(parts, ",")
}

// ── Transition ────────────────────────────────────────────────────────────────

// Transition applies a visual transition effect between a sequence of clips.
// clipPaths must contain at least 2 files.
// Kotlin ref: VideoOptimizerProcessor transition logic.
func (s *EffectsService) Transition(clipPaths []string, transType TransitionType, durationSec float64, outputPath string) error {
	if len(clipPaths) < 2 {
		return fmt.Errorf("transition: need at least 2 clips, got %d", len(clipPaths))
	}
	_ = os.MkdirAll(filepath.Dir(outputPath), 0o755)

	switch transType {
	case TransitionDipBlack:
		return s.transitionDipBlack(clipPaths, durationSec, outputPath)
	default: // xfade and fade both use the xfade filter
		return s.transitionXfade(clipPaths, durationSec, outputPath)
	}
}

// transitionXfade chains FFmpeg xfade filters between consecutive clips.
// FP-005: also chains acrossfade filters for audio to prevent silent output.
func (s *EffectsService) transitionXfade(clipPaths []string, durationSec float64, outputPath string) error {
	// Build -i args
	args := []string{}
	for _, p := range clipPaths {
		args = append(args, "-i", p)
	}

	// Probe durations for accurate offsets.
	durations := s.probeDurations(clipPaths)

	// FP-005: probe each clip for the presence of an audio stream.
	// If none of the clips have audio, skip the audio filter chain entirely.
	hasAudio := s.probeHasAudio(clipPaths)
	anyAudio := false
	for _, a := range hasAudio {
		if a {
			anyAudio = true
			break
		}
	}

	// Build chained xfade filter_complex for video.
	// First pair: [0:v][1:v]xfade=...[x1]
	// Next pair:  [x1][2:v]xfade=...[x2]  ... until N-1 inputs.
	var filterParts []string
	offset := durations[0] - durationSec
	if offset < 0 {
		offset = 0
	}

	prev := "[0:v]"
	for i := 1; i < len(clipPaths); i++ {
		outLabel := fmt.Sprintf("[xv%d]", i)
		if i == len(clipPaths)-1 {
			outLabel = "[vout]"
		}
		f := fmt.Sprintf(
			"%s[%d:v]xfade=transition=fade:duration=%.3f:offset=%.3f%s",
			prev, i, durationSec, offset, outLabel,
		)
		filterParts = append(filterParts, f)
		prev = outLabel
		if i < len(durations) {
			offset += durations[i] - durationSec
			if offset < 0 {
				offset = 0
			}
		} else {
			offset += 10.0 - durationSec // fallback: assume 10s clip
		}
	}

	// FP-005: Build chained acrossfade filter for audio when clips have audio.
	// For clips without audio we use [i:a] which FFmpeg will either pass through
	// or skip gracefully. We use optional map (-map [aout]?) so that if the
	// chain produces nothing the output still succeeds (video-only).
	audioOutLabel := ""
	if anyAudio {
		aPrev := "[0:a]"
		for i := 1; i < len(clipPaths); i++ {
			aOutLabel := fmt.Sprintf("[xa%d]", i)
			if i == len(clipPaths)-1 {
				aOutLabel = "[aout]"
			}
			af := fmt.Sprintf(
				"%s[%d:a]acrossfade=d=%.3f:c1=tri:c2=tri%s",
				aPrev, i, durationSec, aOutLabel,
			)
			filterParts = append(filterParts, af)
			aPrev = aOutLabel
		}
		audioOutLabel = "[aout]"
	}

	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "[vout]",
	)
	if audioOutLabel != "" {
		// Use optional map so that if a clip unexpectedly has no audio the
		// mux still succeeds rather than erroring out.
		args = append(args, "-map", audioOutLabel+"?")
		args = append(args, "-ar", "44100")
	}
	args = append(args,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-y", outputPath,
	)

	s.log.Debug("Transition xfade", "op", "transition", "clips", len(clipPaths),
		"duration", durationSec, "audio", anyAudio)
	if _, err := s.exec.Run(s.binPath, args...); err != nil {
		s.log.Error("Transition xfade failed", "err", err)
		return fmt.Errorf("transition xfade: %w", err)
	}
	return nil
}

// probeHasAudio checks each clip for at least one audio stream using ffprobe.
// Returns a bool slice of the same length as clips; defaults to true on error
// (safer: assume audio present so we don't silently drop it).
func (s *EffectsService) probeHasAudio(clips []string) []bool {
	result := make([]bool, len(clips))
	for i, clip := range clips {
		out, err := s.exec.Run("ffprobe",
			"-v", "error",
			"-select_streams", "a",
			"-show_entries", "stream=codec_type",
			"-of", "csv=p=0",
			clip,
		)
		if err != nil {
			// On probe error assume audio is present (conservative).
			s.log.Warn("probeHasAudio failed, assuming audio present",
				"clip", clip, "err", err)
			result[i] = true
			continue
		}
		result[i] = strings.Contains(strings.TrimSpace(string(out)), "audio")
	}
	return result
}

// transitionDipBlack applies fade-out + fade-in through black between clips.
// Encodes each clip with fade-in/fade-out, then concatenates.
func (s *EffectsService) transitionDipBlack(clipPaths []string, durationSec float64, outputPath string) error {
	tmpDir := os.TempDir()
	var tmpFiles []string
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	durations := s.probeDurations(clipPaths)

	// Encode each clip with appropriate fade in/out.
	for i, clip := range clipPaths {
		tmpOut := filepath.Join(tmpDir, fmt.Sprintf("autocut_dip_%d.mp4", i))
		tmpFiles = append(tmpFiles, tmpOut)

		dur := 10.0
		if i < len(durations) {
			dur = durations[i]
		}

		var vf strings.Builder
		if i > 0 {
			// Fade in at start.
			vf.WriteString(fmt.Sprintf("fade=t=in:st=0:d=%.3f", durationSec))
		}
		if i < len(clipPaths)-1 {
			// Fade out at end.
			fadeOutStart := dur - durationSec
			if fadeOutStart < 0 {
				fadeOutStart = 0
			}
			if vf.Len() > 0 {
				vf.WriteString(",")
			}
			vf.WriteString(fmt.Sprintf("fade=t=out:st=%.3f:d=%.3f", fadeOutStart, durationSec))
		}

		clipArgs := []string{"-i", clip}
		if vf.Len() > 0 {
			clipArgs = append(clipArgs, "-vf", vf.String())
		}
		clipArgs = append(clipArgs, "-c:v", "libx264", "-preset", "fast", "-crf", "23",
			"-c:a", "aac", "-b:a", "192k", "-y", tmpOut)

		if _, err := s.exec.Run(s.binPath, clipArgs...); err != nil {
			s.log.Error("transitionDipBlack encode clip failed", "err", err, "clip", i)
			return fmt.Errorf("transition dip_black encode clip %d: %w", i, err)
		}
	}

	// Write concat list and merge.
	listFile := filepath.Join(tmpDir, "autocut_dip_concat.txt")
	var listEntries []string
	for _, f := range tmpFiles {
		abs, _ := filepath.Abs(f)
		listEntries = append(listEntries, fmt.Sprintf("file '%s'", abs))
	}
	if err := os.WriteFile(listFile, []byte(strings.Join(listEntries, "\n")), 0o644); err != nil {
		return fmt.Errorf("transition dip_black write concat list: %w", err)
	}
	defer os.Remove(listFile)

	mergeArgs := []string{
		"-f", "concat", "-safe", "0", "-i", listFile,
		"-c", "copy", "-y", outputPath,
	}
	s.log.Debug("Transition dip_black merge", "op", "transition", "clips", len(clipPaths))
	if _, err := s.exec.Run(s.binPath, mergeArgs...); err != nil {
		s.log.Error("transitionDipBlack merge failed", "err", err)
		return fmt.Errorf("transition dip_black merge: %w", err)
	}
	return nil
}

// probeDurations uses ffprobe to get clip durations in seconds.
// Returns a slice of the same length as clips; uses 10.0 as fallback on error.
func (s *EffectsService) probeDurations(clips []string) []float64 {
	durations := make([]float64, len(clips))
	for i, clip := range clips {
		out, err := s.exec.Run("ffprobe", "-v", "quiet",
			"-show_entries", "format=duration",
			"-of", "csv=p=0", clip)
		if err != nil {
			s.log.Warn("probeDurations failed, using fallback", "clip", clip, "err", err)
			durations[i] = 10.0
			continue
		}
		trimmed := strings.TrimSpace(string(out))
		v, err := strconv.ParseFloat(trimmed, 64)
		if err != nil || v <= 0 {
			durations[i] = 10.0
		} else {
			durations[i] = v
		}
	}
	return durations
}

// ── TextOverlay ───────────────────────────────────────────────────────────────

// TextOverlay burns customisable text onto the video for a specified time range.
// Kotlin ref: TextOverlayProcessor.
func (s *EffectsService) TextOverlay(
	videoPath, text, fontName string,
	fontSize int,
	color string,
	position TextPosition,
	startSec, endSec float64,
	outputPath string,
) error {
	_ = os.MkdirAll(filepath.Dir(outputPath), 0o755)

	x, y := textPositionXY(position)
	escaped := strings.ReplaceAll(text, `'`, `\'`)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("drawtext=text='%s'", escaped))
	if fontName != "" {
		sb.WriteString(fmt.Sprintf(":fontfile=%s", fontName))
	}
	sb.WriteString(fmt.Sprintf(":fontsize=%d", fontSize))
	sb.WriteString(fmt.Sprintf(":fontcolor=%s", color))
	sb.WriteString(fmt.Sprintf(":x=%s:y=%s", x, y))
	sb.WriteString(fmt.Sprintf(":enable='between(t,%.3f,%.3f)'", startSec, endSec))

	args := []string{
		"-i", videoPath,
		"-vf", sb.String(),
		"-c:a", "copy",
		"-y", outputPath,
	}
	s.log.Debug("TextOverlay", "op", "text_overlay", "position", position, "input", videoPath)
	if _, err := s.exec.Run(s.binPath, args...); err != nil {
		s.log.Error("TextOverlay failed", "err", err, "input", videoPath)
		return fmt.Errorf("text-overlay: %w", err)
	}
	return nil
}

// textPositionXY maps a TextPosition to FFmpeg drawtext x/y expressions.
func textPositionXY(pos TextPosition) (x, y string) {
	switch pos {
	case PosTopLeft:
		return "10", "10"
	case PosTopCenter:
		return "(w-text_w)/2", "10"
	case PosTopRight:
		return "w-text_w-10", "10"
	case PosMidLeft:
		return "10", "(h-text_h)/2"
	case PosMidCenter:
		return "(w-text_w)/2", "(h-text_h)/2"
	case PosBottomLeft:
		return "10", "h-text_h-10"
	case PosBottomCenter:
		return "(w-text_w)/2", "h-text_h-10"
	case PosBottomRight:
		return "w-text_w-10", "h-text_h-10"
	default:
		return "(w-text_w)/2", "h-text_h-10"
	}
}

// ── Overlay ───────────────────────────────────────────────────────────────────

// Overlay composites a logo or watermark image onto the video.
// opacity controls alpha (0.0–1.0), scale is a multiplier for the image size.
// Kotlin ref: VideoOverlayProcessor.
func (s *EffectsService) Overlay(
	videoPath, imagePath string,
	position TextPosition,
	opacity, scale float64,
	outputPath string,
) error {
	_ = os.MkdirAll(filepath.Dir(outputPath), 0o755)

	x, y := overlayPositionXY(position)

	// Scale the logo, then overlay it with alpha.
	filterComplex := fmt.Sprintf(
		"[1:v]scale=iw*%.4f:-1[logo];[0:v][logo]overlay=%s:%s:alpha=%.4f[out]",
		scale, x, y, opacity,
	)

	args := []string{
		"-i", videoPath,
		"-i", imagePath,
		"-filter_complex", filterComplex,
		"-map", "[out]",
		"-map", "0:a",
		"-c:a", "copy",
		"-y", outputPath,
	}
	s.log.Debug("Overlay", "op", "overlay", "position", position, "scale", scale, "opacity", opacity)
	if _, err := s.exec.Run(s.binPath, args...); err != nil {
		s.log.Error("Overlay failed", "err", err, "input", videoPath)
		return fmt.Errorf("overlay: %w", err)
	}
	return nil
}

// VideoOverlayVideo composites a video overlay onto a video using a filter_complex.
// Supports chroma key removal (chromaKey: "black", "green", "white", or "" to skip).
// startSec/endSec control when the overlay is visible (endSec=0 means until end of video).
// Kotlin ref: VideoOverlayProcessor.
func (s *EffectsService) VideoOverlayVideo(
	videoPath, overlayPath string,
	pos TextPosition,
	scale float64,
	chromaKey string,
	startSec, endSec float64,
	outputPath string,
) error {
	if scale == 0 {
		scale = 1.0
	}

	x, y := overlayPositionXY(pos)

	// Build overlay stream: scale → optional chroma key → time-bound overlay
	var filterChain strings.Builder
	filterChain.WriteString(fmt.Sprintf("[1:v]scale=iw*%.4f:ih*%.4f[ov_scaled]", scale, scale))

	ovRef := "[ov_scaled]"
	if chromaKey != "" {
		filterChain.WriteString(fmt.Sprintf(";[ov_scaled]chromakey=%s:0.3:0.05[ov_key]", chromaKey))
		ovRef = "[ov_key]"
	}

	enableFilter := ""
	if endSec > 0 {
		enableFilter = fmt.Sprintf(":enable='between(t,%.2f,%.2f)'", startSec, endSec)
	} else if startSec > 0 {
		enableFilter = fmt.Sprintf(":enable='gte(t,%.2f)'", startSec)
	}

	filterChain.WriteString(fmt.Sprintf(";[0:v]%[1]soverlay=%s:%s%s[v]", ovRef, x, y, enableFilter))

	args := []string{
		"-y",
		"-i", videoPath,
		"-i", overlayPath,
		"-filter_complex", filterChain.String(),
		"-map", "[v]",
		"-map", "0:a?",
		"-c:v", "libx264", "-preset", "fast", "-crf", "23",
		"-c:a", "copy",
		outputPath,
	}

	s.log.Debug("VideoOverlayVideo", "op", "video_overlay_video", "position", pos, "scale", scale, "chromaKey", chromaKey)
	if out, err := s.exec.Run(s.binPath, args...); err != nil {
		return fmt.Errorf("ffmpeg video overlay: %w: %s", err, string(out))
	}
	return nil
}

// overlayPositionXY maps a TextPosition to FFmpeg overlay x:y expressions.
// Uses W/H for main video dimensions, w/h for overlay dimensions.
func overlayPositionXY(pos TextPosition) (x, y string) {
	switch pos {
	case PosTopLeft:
		return "10", "10"
	case PosTopCenter:
		return "(W-w)/2", "10"
	case PosTopRight:
		return "W-w-10", "10"
	case PosMidLeft:
		return "10", "(H-h)/2"
	case PosMidCenter:
		return "(W-w)/2", "(H-h)/2"
	case PosBottomLeft:
		return "10", "H-h-10"
	case PosBottomCenter:
		return "(W-w)/2", "H-h-10"
	case PosBottomRight:
		return "W-w-10", "H-h-10"
	default:
		return "(W-w)/2", "H-h-10"
	}
}
