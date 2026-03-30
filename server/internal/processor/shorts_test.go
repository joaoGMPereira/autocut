package processor

import (
	"strings"
	"testing"
)

// TestGenerate_DefaultConfig_CropDimensions verifies that Generate with default
// ShortsConfig produces the expected crop=1080:1920 filter in the ffmpeg args.
// Kotlin ref: VerticalVideoProcessor.buildFfmpegCommand — default WIDTH=1080, HEIGHT=1920.
func TestGenerate_DefaultConfig_CropDimensions(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	cfg := ShortsConfig{} // zero-value triggers defaults: 1080x1920
	_ = gen.Generate("/input.mp4", cfg, "/dev/null/out.mp4")

	args := mock.lastArgs()
	if args == nil {
		t.Fatal("no ffmpeg call recorded")
	}

	// Find -vf value
	var vf string
	for i, a := range args {
		if a == "-vf" && i+1 < len(args) {
			vf = args[i+1]
			break
		}
	}
	if vf == "" {
		t.Fatalf("no -vf flag in args: %v", args)
	}

	// crop=1080:1920 must appear in the filter
	if !strings.Contains(vf, "crop=1080:1920") {
		t.Errorf("expected crop=1080:1920 in vf filter, got: %s", vf)
	}
}

func TestGenerate_DefaultConfig_ScaleDimensions(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	cfg := ShortsConfig{}
	_ = gen.Generate("/input.mp4", cfg, "/dev/null/out.mp4")

	args := mock.lastArgs()
	var vf string
	for i, a := range args {
		if a == "-vf" && i+1 < len(args) {
			vf = args[i+1]
			break
		}
	}
	// scale=1080:1920 must also appear (after crop)
	if !strings.Contains(vf, "scale=1080:1920") {
		t.Errorf("expected scale=1080:1920 in vf filter, got: %s", vf)
	}
}

func TestGenerate_CustomDimensions(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	cfg := ShortsConfig{Width: 720, Height: 1280}
	_ = gen.Generate("/input.mp4", cfg, "/dev/null/out.mp4")

	args := mock.lastArgs()
	var vf string
	for i, a := range args {
		if a == "-vf" && i+1 < len(args) {
			vf = args[i+1]
			break
		}
	}
	if !strings.Contains(vf, "crop=720:1280") {
		t.Errorf("expected crop=720:1280 for custom dims, got: %s", vf)
	}
}

func TestGenerate_CropOffsets(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	cfg := ShortsConfig{Width: 1080, Height: 1920, CropX: 100, CropY: 50}
	_ = gen.Generate("/input.mp4", cfg, "/dev/null/out.mp4")

	args := mock.lastArgs()
	var vf string
	for i, a := range args {
		if a == "-vf" && i+1 < len(args) {
			vf = args[i+1]
			break
		}
	}
	// crop=w:h:x:y — offsets must be present
	if !strings.Contains(vf, "crop=1080:1920:100:50") {
		t.Errorf("expected crop=1080:1920:100:50 with offsets, got: %s", vf)
	}
}

func TestGenerate_ContainsFaststart(t *testing.T) {
	mock := &MockExecutor{}
	gen := newShortsGeneratorWithExecutor("ffmpeg", mock)

	_ = gen.Generate("/input.mp4", ShortsConfig{}, "/dev/null/out.mp4")

	args := mock.lastArgs()
	if !containsSeq(args, "-movflags", "+faststart") {
		t.Errorf("expected -movflags +faststart for Shorts, got: %v", args)
	}
}
