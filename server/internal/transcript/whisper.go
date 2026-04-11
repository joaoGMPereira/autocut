package transcript

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/toolerr"
)

// WhisperTranscriber runs whisper.cpp against audio/video files.
// For videos <= ChunkDuration it calls whisper once.
// For longer videos it splits via ffmpeg, transcribes each chunk,
// and merges segments with adjusted timestamps.
//
// Kotlin ref: TranscriptGenerator + TranscriptWhisperDelegate + TranscriptChunkDelegate
type WhisperTranscriber struct {
	cfg  WhisperConfig
	exec Executor
	log  *slog.Logger
}

// New creates a WhisperTranscriber with the provided config.
// Defaults are applied for zero fields (BinPath, ChunkDuration, Language).
func New(cfg WhisperConfig) *WhisperTranscriber {
	cfg.defaults()
	return &WhisperTranscriber{
		cfg:  cfg,
		exec: DefaultExecutor{},
		log:  slog.With("component", "transcript"),
	}
}

// newWithExecutor is used in tests to inject a mock executor.
func newWithExecutor(cfg WhisperConfig, ex Executor) *WhisperTranscriber {
	cfg.defaults()
	return &WhisperTranscriber{
		cfg:  cfg,
		exec: ex,
		log:  slog.With("component", "transcript"),
	}
}

// Transcribe transcribes the audio/video at audioPath.
// For files whose duration exceeds cfg.ChunkDuration, it chunks via ffmpeg.
func (w *WhisperTranscriber) Transcribe(audioPath string) (*Transcript, error) {
	dur, err := w.probeDuration(audioPath)
	if err != nil {
		// If we cannot probe (e.g. in tests), treat as short video.
		w.log.Warn("could not probe duration, assuming short video",
			"component", "transcript", "err", err, "path", audioPath)
		dur = 0
	}

	if dur <= w.cfg.ChunkDuration {
		return w.transcribeSingle(audioPath, 0)
	}
	return w.transcribeChunked(audioPath, dur)
}

// transcribeSingle calls whisper on audioPath and returns the parsed transcript.
// offset is added to all segment timestamps (used when processing chunks).
func (w *WhisperTranscriber) transcribeSingle(audioPath string, offset time.Duration) (*Transcript, error) {
	tmpDir, err := os.MkdirTemp("", "whisper-*")
	if err != nil {
		return nil, fmt.Errorf("transcribeSingle: create tmpDir: %w", err)
	}
	defer func() {
		if rerr := os.RemoveAll(tmpDir); rerr != nil {
			w.log.Warn("cleanup tmpDir failed", "err", rerr)
		}
	}()

	// Compute stem from the original input for output-JSON lookup.
	base := filepath.Base(audioPath)
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]

	// whisper.cpp (standard Homebrew build) cannot decode MP4/MKV/etc.
	// Extract to 16 kHz mono PCM WAV so whisper can always read the input.
	audioForWhisper := audioPath
	if strings.ToLower(ext) != ".wav" {
		wavPath := filepath.Join(tmpDir, stem+".wav")
		if _, err := w.exec.Run("ffmpeg",
			"-i", audioPath, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", "-y", wavPath,
		); err != nil {
			return nil, &toolerr.ToolErr{Tool: "ffmpeg", Op: "extract audio", Cause: err}
		}
		if fi, e := os.Stat(wavPath); e == nil {
			w.log.Info("audio extracted to WAV", "op", "transcript", "path", wavPath, "size", fi.Size())
		}
		audioForWhisper = wavPath
	}

	args := w.buildWhisperArgs(audioForWhisper, tmpDir)
	w.log.Info("running whisper", "op", "transcript", "path", audioPath, "offset", offset)

	out, err := w.exec.Run(w.cfg.BinPath, args...)
	w.log.Debug("whisper output", "op", "transcript", "output", string(out))
	if err != nil {
		return nil, &toolerr.ToolErr{Tool: "whisper", Op: "transcribe", Cause: err}
	}

	// whisper writes {stem}.json inside --output-dir
	// For non-WAV: audioForWhisper = tmpDir/stem.wav → whisper writes tmpDir/stem.json ✓
	// For WAV:     audioForWhisper = audioPath        → whisper writes tmpDir/stem.json ✓
	jsonPath := filepath.Join(tmpDir, stem+".json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		// Log all files in tmpDir to diagnose unexpected output naming.
		if entries, de := os.ReadDir(tmpDir); de == nil {
			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.Name()
			}
			w.log.Error("tmpDir contents after whisper", "files", names)
		}
		return nil, fmt.Errorf("transcribeSingle: read output JSON %q: %w", jsonPath, err)
	}

	t, err := ParseWhisperJSON(data)
	if err != nil {
		return nil, fmt.Errorf("transcribeSingle: parse JSON: %w", err)
	}

	if offset > 0 {
		applyOffset(t, offset)
	}

	return t, nil
}

// transcribeChunked splits audioPath into chunks of w.cfg.ChunkDuration via
// ffmpeg, transcribes each, adjusts timestamps, and merges all segments.
//
// ffmpeg command per chunk:
//
//	ffmpeg -i audioPath -ss <offset> -t <duration> -c copy <chunkPath>
func (w *WhisperTranscriber) transcribeChunked(audioPath string, totalDur time.Duration) (*Transcript, error) {
	w.log.Info("chunking long video",
		"op", "transcript", "totalDur", totalDur, "chunkDur", w.cfg.ChunkDuration)

	chunkDur := w.cfg.ChunkDuration
	numChunks := int(totalDur/chunkDur) + 1
	if totalDur%chunkDur == 0 {
		numChunks = int(totalDur / chunkDur)
	}

	tmpDir, err := os.MkdirTemp("", "whisper-chunks-*")
	if err != nil {
		return nil, fmt.Errorf("transcribeChunked: create tmpDir: %w", err)
	}
	defer func() {
		if rerr := os.RemoveAll(tmpDir); rerr != nil {
			w.log.Warn("cleanup chunks tmpDir failed", "err", rerr)
		}
	}()

	var allSegments []Segment
	var language string

	for i := 0; i < numChunks; i++ {
		offset := time.Duration(i) * chunkDur
		remaining := totalDur - offset
		dur := chunkDur
		if remaining < dur {
			dur = remaining
		}
		if dur <= 0 {
			break
		}

		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%03d%s", i, filepath.Ext(audioPath)))
		ffArgs := buildFFmpegChunkArgs(audioPath, offset, dur, chunkPath)

		w.log.Info("extracting chunk",
			"op", "transcript", "chunk", i+1, "of", numChunks,
			"offset", offset, "dur", dur)

		if _, err := w.exec.Run("ffmpeg", ffArgs...); err != nil {
			return nil, &toolerr.ToolErr{Tool: "ffmpeg", Op: "chunk audio", Cause: err}
		}

		chunkTranscript, err := w.transcribeSingle(chunkPath, offset)
		if err != nil {
			return nil, fmt.Errorf("transcribeChunked: transcribe chunk %d: %w", i, err)
		}

		allSegments = append(allSegments, chunkTranscript.Segments...)
		if language == "" && chunkTranscript.Language != "" {
			language = chunkTranscript.Language
		}
	}

	return &Transcript{
		Segments: allSegments,
		Language: language,
		Duration: totalDur,
	}, nil
}

// buildWhisperArgs constructs the argument slice for the whisper binary.
// Uses --output-file <dir/stem> (without extension) so whisper writes <dir/stem>.json.
// Note: --output-dir is NOT supported by the Homebrew whisper-cpp build.
// Kotlin ref: TranscriptWhisperDelegate.buildWhisperArgs
func (w *WhisperTranscriber) buildWhisperArgs(audioPath, outputDir string) []string {
	base := filepath.Base(audioPath)
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	outputBase := filepath.Join(outputDir, stem)

	args := []string{audioPath}
	if w.cfg.ModelPath != "" {
		args = append(args, "--model", w.cfg.ModelPath)
	}
	if w.cfg.Language != "" {
		args = append(args, "--language", w.cfg.Language)
	}
	args = append(args, "--output-json", "--output-file", outputBase)
	return args
}

// buildFFmpegChunkArgs builds the ffmpeg argument slice for a single chunk.
// Kotlin ref: TranscriptAudioDelegate.extractAudioChunk (uses -ss/-t/-c copy)
func buildFFmpegChunkArgs(input string, offset, dur time.Duration, output string) []string {
	return []string{
		"-i", input,
		"-ss", formatDurationSecs(offset),
		"-t", formatDurationSecs(dur),
		"-c", "copy",
		"-y",
		output,
	}
}

// probeDuration uses ffprobe to read the media duration in seconds.
func (w *WhisperTranscriber) probeDuration(path string) (time.Duration, error) {
	out, err := w.exec.Run("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		path,
	)
	if err != nil {
		return 0, fmt.Errorf("probeDuration: ffprobe: %w", err)
	}

	var probe struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		return 0, fmt.Errorf("probeDuration: parse ffprobe output: %w", err)
	}
	secs, err := strconv.ParseFloat(probe.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("probeDuration: parse duration %q: %w", probe.Format.Duration, err)
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// applyOffset shifts all segment timestamps by the given offset.
func applyOffset(t *Transcript, offset time.Duration) {
	for i := range t.Segments {
		t.Segments[i].Start += offset
		t.Segments[i].End += offset
	}
}

// formatDurationSecs formats a time.Duration as a float64 seconds string for ffmpeg.
func formatDurationSecs(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}
