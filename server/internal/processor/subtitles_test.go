package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestGenerateSRT
// ---------------------------------------------------------------------------

// TestGenerateSRT verifies that GenerateSRT writes a correctly formatted SRT file.
func TestGenerateSRT(t *testing.T) {
	segments := []SubtitleSegment{
		{Start: 0, End: 1500 * time.Millisecond, Text: "Hello"},
		{Start: 2000 * time.Millisecond, End: 3500 * time.Millisecond, Text: "World"},
		{Start: 4000 * time.Millisecond, End: 5000 * time.Millisecond, Text: "!"},
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "test.srt")

	g := NewSubtitleGenerator("")
	if err := g.GenerateSRT(segments, outPath); err != nil {
		t.Fatalf("GenerateSRT returned error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read SRT output: %v", err)
	}
	content := string(data)

	// Check block 1
	if !strings.Contains(content, "1\n") {
		t.Error("expected line '1' for first entry")
	}
	if !strings.Contains(content, "00:00:00,000 --> 00:00:01,500") {
		t.Errorf("expected first timestamp, got:\n%s", content)
	}
	if !strings.Contains(content, "Hello") {
		t.Error("expected text 'Hello' in SRT output")
	}

	// Check block 2
	if !strings.Contains(content, "2\n") {
		t.Error("expected line '2' for second entry")
	}
	if !strings.Contains(content, "00:00:02,000 --> 00:00:03,500") {
		t.Errorf("expected second timestamp, got:\n%s", content)
	}
	if !strings.Contains(content, "World") {
		t.Error("expected text 'World' in SRT output")
	}

	// Check block 3
	if !strings.Contains(content, "!") {
		t.Error("expected text '!' in SRT output")
	}
}

// ---------------------------------------------------------------------------
// TestTimestampFormat
// ---------------------------------------------------------------------------

// TestTimestampFormat verifies formatSRTTimestamp converts durations correctly.
func TestTimestampFormat(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{
			d:    time.Hour + time.Minute + time.Second + 500*time.Millisecond,
			want: "01:01:01,500",
		},
		{
			d:    0,
			want: "00:00:00,000",
		},
		{
			d:    59*time.Second + 999*time.Millisecond,
			want: "00:00:59,999",
		},
		{
			d:    90 * time.Minute,
			want: "01:30:00,000",
		},
	}

	for _, c := range cases {
		got := formatSRTTimestamp(c.d)
		if got != c.want {
			t.Errorf("formatSRTTimestamp(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBurnSubtitles
// ---------------------------------------------------------------------------

// TestBurnSubtitles verifies BurnSubtitles assembles correct ffmpeg arguments.
func TestBurnSubtitles(t *testing.T) {
	mock := &MockExecutor{}
	g := newSubtitleGeneratorWithExecutor("ffmpeg", mock)

	err := g.BurnSubtitles("video.mp4", "subs.srt", "out.mp4", SubtitleConfig{})
	if err != nil {
		t.Fatalf("BurnSubtitles returned unexpected error: %v", err)
	}

	if len(mock.calls) == 0 {
		t.Fatal("expected ffmpeg to be called")
	}

	args := mock.allArgs()

	// Must include video input
	if !containsArg(args, "video.mp4") {
		t.Error("expected video.mp4 in args")
	}

	// Must include the vf filter with "subtitles="
	found := false
	for _, a := range args {
		if strings.Contains(a, "subtitles=subs.srt") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'subtitles=subs.srt' in args, got: %v", args)
	}

	// Must include force_style with FontName/FontSize
	found = false
	for _, a := range args {
		if strings.Contains(a, "force_style") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'force_style' in args, got: %v", args)
	}
}

// ---------------------------------------------------------------------------
// TestDefaultSubtitleConfig
// ---------------------------------------------------------------------------

// TestDefaultSubtitleConfig verifies that defaultSubtitleConfig fills in zero values.
func TestDefaultSubtitleConfig(t *testing.T) {
	cfg := defaultSubtitleConfig(SubtitleConfig{})
	if cfg.FontName != "Arial" {
		t.Errorf("expected FontName 'Arial', got %q", cfg.FontName)
	}
	if cfg.FontSize != 20 {
		t.Errorf("expected FontSize 20, got %d", cfg.FontSize)
	}
	if cfg.FontColor != "white" {
		t.Errorf("expected FontColor 'white', got %q", cfg.FontColor)
	}
	if cfg.Position != "bottom" {
		t.Errorf("expected Position 'bottom', got %q", cfg.Position)
	}
}

// ---------------------------------------------------------------------------
// TestGenerateAndBurn
// ---------------------------------------------------------------------------

// TestGenerateAndBurn verifies the combined pipeline calls ffmpeg exactly once.
func TestGenerateAndBurn(t *testing.T) {
	mock := &MockExecutor{}
	g := newSubtitleGeneratorWithExecutor("ffmpeg", mock)

	segments := []SubtitleSegment{
		{Start: 0, End: 1 * time.Second, Text: "Test"},
	}

	if err := g.GenerateAndBurn("video.mp4", segments, "out.mp4", SubtitleConfig{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// GenerateAndBurn: GenerateSRT writes directly (no exec), BurnSubtitles calls exec once
	if len(mock.calls) != 1 {
		t.Errorf("expected 1 ffmpeg call, got %d", len(mock.calls))
	}
}
