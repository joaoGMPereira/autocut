package processor

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// AC01 — Blur background composite
// ---------------------------------------------------------------------------

// TestGenerateBlur_FilterComplex verifies that Generate with AddBlurBackground=true
// uses -filter_complex (not -vf) and includes boxblur=20:20 in the filter.
// Kotlin ref: VerticalVideoProcessor.buildBlurCompositeFilter (LL-005).
func TestGenerateBlur_FilterComplex(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	cfg := ShortsConfig{AddBlurBackground: true}
	_ = gen.Generate("/input.mp4", cfg, "/dev/null/out.mp4")

	args := mock.lastArgs()
	if args == nil {
		t.Fatal("no ffmpeg call recorded")
	}

	// Must use -filter_complex, not -vf
	if containsArg(args, "-vf") {
		t.Error("blur path must not use -vf; expected -filter_complex")
	}
	if !containsArg(args, "-filter_complex") {
		t.Error("blur path must use -filter_complex")
	}

	// Find filter_complex value
	fc := filterComplexValue(args)
	if fc == "" {
		t.Fatal("no -filter_complex value found")
	}

	if !strings.Contains(fc, "boxblur=20:20") {
		t.Errorf("expected boxblur=20:20 in filter_complex, got: %s", fc)
	}
	if !strings.Contains(fc, "overlay=(W-w)/2:(H-h)/2") {
		t.Errorf("expected overlay=(W-w)/2:(H-h)/2 in filter_complex, got: %s", fc)
	}
}

// TestGenerateBlur_DefaultDimensions verifies that 1080x1920 defaults are used
// in the blur path when no dimensions are set.
func TestGenerateBlur_DefaultDimensions(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	cfg := ShortsConfig{AddBlurBackground: true}
	_ = gen.Generate("/input.mp4", cfg, "/dev/null/out.mp4")

	fc := filterComplexValue(mock.lastArgs())
	if !strings.Contains(fc, "1080") || !strings.Contains(fc, "1920") {
		t.Errorf("expected 1080x1920 in blur filter, got: %s", fc)
	}
}

// TestGenerateBlur_MapVideoAudio verifies that -map [v] and -map 0:a? are present.
func TestGenerateBlur_MapVideoAudio(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	_ = gen.Generate("/input.mp4", ShortsConfig{AddBlurBackground: true}, "/dev/null/out.mp4")
	args := mock.lastArgs()

	if !containsSeq(args, "-map", "[v]") {
		t.Errorf("expected -map [v] in blur args, got: %v", args)
	}
}

// TestGenerateNonBlur_StillUsesVf verifies backward-compat: no AddBlurBackground
// still uses the original -vf crop path.
func TestGenerateNonBlur_StillUsesVf(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	_ = gen.Generate("/input.mp4", ShortsConfig{}, "/dev/null/out.mp4")
	args := mock.lastArgs()

	if !containsArg(args, "-vf") {
		t.Error("non-blur path must use -vf")
	}
	if containsArg(args, "-filter_complex") {
		t.Error("non-blur path must not use -filter_complex")
	}
}

// ---------------------------------------------------------------------------
// AC02 — Word-by-word segment splitting
// ---------------------------------------------------------------------------

// TestTranscriptToWordSegments_TwoWords verifies proportional splitting for a
// 2-word segment spanning 1 second: each word gets ~0.5s.
func TestTranscriptToWordSegments_TwoWords(t *testing.T) {
	segs := []SubtitleSegment{
		{Start: 0, End: time.Second, Text: "hello world"},
	}
	result := transcriptToWordSegments(segs)

	if len(result) != 2 {
		t.Fatalf("expected 2 word segments, got %d", len(result))
	}
	if result[0].Text != "hello" {
		t.Errorf("expected 'hello', got %q", result[0].Text)
	}
	if result[1].Text != "world" {
		t.Errorf("expected 'world', got %q", result[1].Text)
	}
	// First word: [0, 500ms], second: [500ms, 1s]
	if result[0].Start != 0 {
		t.Errorf("first word start should be 0, got %v", result[0].Start)
	}
	if result[0].End != 500*time.Millisecond {
		t.Errorf("first word end should be 500ms, got %v", result[0].End)
	}
	if result[1].Start != 500*time.Millisecond {
		t.Errorf("second word start should be 500ms, got %v", result[1].Start)
	}
	if result[1].End != time.Second {
		t.Errorf("second word end should be 1s, got %v", result[1].End)
	}
}

// TestTranscriptToWordSegments_SingleWord verifies a single-word segment passes through.
func TestTranscriptToWordSegments_SingleWord(t *testing.T) {
	segs := []SubtitleSegment{
		{Start: 0, End: 2 * time.Second, Text: "hello"},
	}
	result := transcriptToWordSegments(segs)
	if len(result) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(result))
	}
	if result[0].Text != "hello" {
		t.Errorf("expected 'hello', got %q", result[0].Text)
	}
	if result[0].Start != 0 || result[0].End != 2*time.Second {
		t.Errorf("single word should preserve full range, got [%v, %v]", result[0].Start, result[0].End)
	}
}

// TestTranscriptToWordSegments_EmptyTextSkipped verifies segments with no words
// are dropped.
func TestTranscriptToWordSegments_EmptyTextSkipped(t *testing.T) {
	segs := []SubtitleSegment{
		{Start: 0, End: time.Second, Text: "   "},
	}
	result := transcriptToWordSegments(segs)
	if len(result) != 0 {
		t.Errorf("expected 0 segments for whitespace-only text, got %d", len(result))
	}
}

// TestTranscriptToWordSegments_MultipleSegments verifies that multiple input
// segments each get their own per-word entries independently.
func TestTranscriptToWordSegments_MultipleSegments(t *testing.T) {
	segs := []SubtitleSegment{
		{Start: 0, End: time.Second, Text: "a b"},
		{Start: 2 * time.Second, End: 3 * time.Second, Text: "c d e"},
	}
	result := transcriptToWordSegments(segs)
	if len(result) != 5 { // 2 + 3
		t.Fatalf("expected 5 word segments, got %d", len(result))
	}
}

// TestTranscriptToWordSegments_LastWordEndsAtSegmentEnd verifies the last word
// snaps to the segment's exact end time (handles rounding remainder).
func TestTranscriptToWordSegments_LastWordEndsAtSegmentEnd(t *testing.T) {
	segs := []SubtitleSegment{
		{Start: 0, End: time.Second, Text: "a b c"},
	}
	result := transcriptToWordSegments(segs)
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	if result[2].End != time.Second {
		t.Errorf("last word end should be 1s, got %v", result[2].End)
	}
}

// ---------------------------------------------------------------------------
// Helpers used only in this file
// ---------------------------------------------------------------------------

// filterComplexValue extracts the value following -filter_complex from args.
func filterComplexValue(args []string) string {
	for i, a := range args {
		if a == "-filter_complex" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
