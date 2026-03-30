package transcript

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockExecutor records calls and returns pre-configured responses.
// It also simulates whisper by writing a JSON file into the output-dir argument.
type mockExecutor struct {
	calls []mockCall
	// ffprobeResp is the JSON returned when "ffprobe" is called.
	ffprobeResp []byte
	// err to return on any call (nil = success)
	err error
}

type mockCall struct {
	name string
	args []string
}

// Run records the call and, for whisper, creates the expected output JSON.
func (m *mockExecutor) Run(name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, mockCall{name: name, args: args})

	if m.err != nil {
		return nil, m.err
	}

	switch name {
	case "ffprobe":
		if m.ffprobeResp != nil {
			return m.ffprobeResp, nil
		}
		return nil, fmt.Errorf("mockExecutor: ffprobe not configured")

	case "ffmpeg":
		// Simulate ffmpeg by creating an empty chunk file at the output path (last arg).
		outPath := args[len(args)-1]
		if err := os.WriteFile(outPath, []byte("fake-audio"), 0o644); err != nil {
			return nil, fmt.Errorf("mockExecutor: create chunk file: %w", err)
		}
		return nil, nil

	default:
		// Assume it's the whisper binary: locate --output-dir and write JSON.
		return nil, m.handleWhisper(name, args)
	}
}

// handleWhisper finds --output-dir in args and writes a minimal whisper JSON.
func (m *mockExecutor) handleWhisper(_ string, args []string) error {
	outputDir := ""
	audioPath := ""
	for i, a := range args {
		if a == "--output-dir" && i+1 < len(args) {
			outputDir = args[i+1]
		}
		// First arg is the audio path (before any flags)
		if i == 0 {
			audioPath = a
		}
	}
	if outputDir == "" {
		return fmt.Errorf("mockExecutor: --output-dir not found in whisper args %v", args)
	}

	seg := whisperSegmentJSON{
		Start: 0.0,
		End:   2.0,
		Text:  "mock transcription",
	}
	out := whisperOutputJSON{Segments: []whisperSegmentJSON{seg}}
	data, _ := json.Marshal(out)

	base := filepath.Base(audioPath)
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	jsonPath := filepath.Join(outputDir, stem+".json")
	return os.WriteFile(jsonPath, data, 0o644)
}

// ffprobeJSON builds a minimal ffprobe JSON response for a given duration in seconds.
func ffprobeJSON(durationSecs float64) []byte {
	s := fmt.Sprintf(`{"format":{"duration":"%g"}}`, durationSecs)
	return []byte(s)
}

// countWhisperCalls returns the number of times the whisper binary was called.
func countWhisperCalls(m *mockExecutor, binPath string) int {
	n := 0
	for _, c := range m.calls {
		if c.name == binPath {
			n++
		}
	}
	return n
}

// TestTranscribeShortVideo verifies that for a file under ChunkDuration,
// whisper is called exactly once and ffmpeg is not called for chunking.
func TestTranscribeShortVideo(t *testing.T) {
	// 10-minute video — below the 25-min default
	mock := &mockExecutor{
		ffprobeResp: ffprobeJSON(600), // 600s = 10min
	}

	cfg := WhisperConfig{
		BinPath:       "whisper-mock",
		ModelPath:     "/models/base.bin",
		Language:      "en",
		ChunkDuration: 25 * time.Minute,
	}
	tr := newWithExecutor(cfg, mock)

	// Create a temp audio file so the stem-based JSON lookup resolves.
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "short_video.wav")
	if err := os.WriteFile(audioPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := tr.Transcribe(audioPath)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil transcript")
	}

	whisperCalls := countWhisperCalls(mock, "whisper-mock")
	if whisperCalls != 1 {
		t.Errorf("expected 1 whisper call for short video, got %d", whisperCalls)
	}

	// No ffmpeg chunking calls (only ffprobe).
	for _, c := range mock.calls {
		if c.name == "ffmpeg" {
			t.Errorf("unexpected ffmpeg call for short video: %v", c.args)
		}
	}
}

// TestTranscribeChunking verifies that a long video results in multiple
// ffmpeg chunk extractions and multiple whisper calls.
func TestTranscribeChunking(t *testing.T) {
	// 3600s = 60min → with 25min chunks → 3 chunks expected
	mock := &mockExecutor{
		ffprobeResp: ffprobeJSON(3600),
	}

	cfg := WhisperConfig{
		BinPath:       "whisper-mock",
		ModelPath:     "/models/base.bin",
		Language:      "pt",
		ChunkDuration: 25 * time.Minute,
	}
	tr := newWithExecutor(cfg, mock)

	dir := t.TempDir()
	audioPath := filepath.Join(dir, "long_video.wav")
	if err := os.WriteFile(audioPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := tr.Transcribe(audioPath)
	if err != nil {
		t.Fatalf("Transcribe chunked failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil transcript")
	}

	// 3600s / (25*60)s = 2.4 → 3 chunks
	expectedChunks := 3
	ffmpegCalls := 0
	for _, c := range mock.calls {
		if c.name == "ffmpeg" {
			ffmpegCalls++
			// Verify -ss and -t args are present.
			joined := strings.Join(c.args, " ")
			if !strings.Contains(joined, "-ss") || !strings.Contains(joined, "-t") {
				t.Errorf("ffmpeg chunk call missing -ss/-t: %v", c.args)
			}
		}
	}
	if ffmpegCalls != expectedChunks {
		t.Errorf("expected %d ffmpeg chunk calls, got %d", expectedChunks, ffmpegCalls)
	}

	whisperCalls := countWhisperCalls(mock, "whisper-mock")
	if whisperCalls != expectedChunks {
		t.Errorf("expected %d whisper calls, got %d", expectedChunks, whisperCalls)
	}

	// Segments from all chunks should be present.
	if len(result.Segments) != expectedChunks {
		t.Errorf("expected %d merged segments (1 per chunk), got %d",
			expectedChunks, len(result.Segments))
	}

	// Verify offset was applied: second chunk segment should start >= 25min.
	if len(result.Segments) >= 2 {
		chunkDur := 25 * time.Minute
		if result.Segments[1].Start < chunkDur {
			t.Errorf("chunk 2 offset not applied: Start=%v, want >= %v",
				result.Segments[1].Start, chunkDur)
		}
	}
}
