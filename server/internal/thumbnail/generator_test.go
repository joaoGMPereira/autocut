package thumbnail

import (
	"fmt"
	"strings"
	"testing"
)

// MockExecutor records every invocation so tests can inspect what was called.
type MockExecutor struct {
	Calls  []MockCall
	Output []byte
	Err    error
}

// MockCall records a single invocation of Run.
type MockCall struct {
	Name string
	Args []string
}

func (m *MockExecutor) Run(name string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, MockCall{Name: name, Args: args})
	return m.Output, m.Err
}

// argsContain returns true when any element in args contains the substring sub.
func argsContain(args []string, sub string) bool {
	for _, a := range args {
		if strings.Contains(a, sub) {
			return true
		}
	}
	return false
}

// TestGenerateBrandedFFmpeg verifies that GenerateBranded without a FontPath
// calls ffmpeg with a "drawtext" filter.
func TestGenerateBrandedFFmpeg(t *testing.T) {
	mock := &MockExecutor{}
	g := newWithExec("ffmpeg", "convert", mock)

	cfg := BrandedConfig{
		BackgroundPath: "/tmp/bg.jpg",
		TextOverlay:    "Hello World",
		FontColor:      "white",
		FontSize:       80,
		Width:          1280,
		Height:         720,
		Position:       Position{Anchor: AnchorCenter},
	}

	// The mock returns no output; the call will "fail" from ffmpeg's perspective
	// but we only care that the right binary and args were assembled.
	_ = g.GenerateBranded(cfg, "/tmp/out.jpg")

	if len(mock.Calls) == 0 {
		t.Fatal("expected ffmpeg to be called, got 0 calls")
	}

	call := mock.Calls[0]
	if call.Name != "ffmpeg" {
		t.Errorf("expected binary %q, got %q", "ffmpeg", call.Name)
	}
	if !argsContain(call.Args, "drawtext") {
		t.Errorf("expected 'drawtext' in ffmpeg args, got: %v", call.Args)
	}
	if !argsContain(call.Args, "/tmp/bg.jpg") {
		t.Errorf("expected background path in args, got: %v", call.Args)
	}
}

// TestGenerateBrandedMagick verifies that when FontPath is set the generator
// calls ImageMagick (convert) instead of ffmpeg.
func TestGenerateBrandedMagick(t *testing.T) {
	mock := &MockExecutor{}
	g := newWithExec("ffmpeg", "convert", mock)

	cfg := BrandedConfig{
		BackgroundPath: "/tmp/bg.jpg",
		TextOverlay:    "Hello Magick",
		FontPath:       "/Library/Fonts/Arial.ttf",
		FontColor:      "white",
		FontSize:       80,
		Width:          1280,
		Height:         720,
	}

	_ = g.GenerateBranded(cfg, "/tmp/out.jpg")

	if len(mock.Calls) == 0 {
		t.Fatal("expected a command to be called, got 0 calls")
	}
	if mock.Calls[0].Name != "convert" {
		t.Errorf("expected binary %q, got %q", "convert", mock.Calls[0].Name)
	}
}

// TestGenerateCentered verifies that GenerateCentered produces a -filter_complex
// call with "(w-text_w)/2" for the text x-position.
func TestGenerateCentered(t *testing.T) {
	mock := &MockExecutor{}
	g := newWithExec("ffmpeg", "convert", mock)

	cfg := CenteredConfig{
		FramePath:   "/tmp/frame.jpg",
		TextOverlay: "My Short",
		FontColor:   "white",
		FontSize:    70,
		Width:       1280,
		Height:      720,
	}

	_ = g.GenerateCentered(cfg, "/tmp/short.jpg")

	if len(mock.Calls) == 0 {
		t.Fatal("expected ffmpeg to be called, got 0 calls")
	}

	call := mock.Calls[0]
	if call.Name != "ffmpeg" {
		t.Errorf("expected binary %q, got %q", "ffmpeg", call.Name)
	}
	if !argsContain(call.Args, "-filter_complex") {
		t.Errorf("expected '-filter_complex' in args, got: %v", call.Args)
	}
	if !argsContain(call.Args, "(w-text_w)/2") {
		t.Errorf("expected '(w-text_w)/2' in filter, got: %v", call.Args)
	}
}

// TestPreview verifies that Preview calls ffmpeg with "scale=320:180".
func TestPreview(t *testing.T) {
	mock := &MockExecutor{}
	g := newWithExec("ffmpeg", "convert", mock)

	_ = g.Preview("/tmp/thumb.jpg", "/tmp/preview.jpg")

	if len(mock.Calls) == 0 {
		t.Fatal("expected ffmpeg to be called, got 0 calls")
	}

	call := mock.Calls[0]
	if call.Name != "ffmpeg" {
		t.Errorf("expected binary %q, got %q", "ffmpeg", call.Name)
	}
	if !argsContain(call.Args, "scale=320:180") {
		t.Errorf("expected 'scale=320:180' in args, got: %v", call.Args)
	}
}

// TestExtractBestFrameUsesSeek verifies that when ffprobe fails the generator
// falls back to a 0-second seek and still calls ffmpeg.
func TestExtractBestFrameUsesSeek(t *testing.T) {
	te := &trackingExec{
		runs: []runResult{
			{out: nil, err: fmt.Errorf("ffprobe not found")}, // ffprobe → error
			{out: []byte(""), err: nil},                       // ffmpeg frame extract
		},
	}
	g := newWithExec("ffmpeg", "convert", te)

	_ = g.ExtractBestFrame("/tmp/video.mp4", "/tmp/frame.jpg")

	if len(te.calls) < 2 {
		t.Fatalf("expected at least 2 exec calls (ffprobe + ffmpeg), got %d", len(te.calls))
	}

	ffmpegCall := te.calls[1]
	if ffmpegCall.Name != "ffmpeg" {
		t.Errorf("second call should be ffmpeg, got %q", ffmpegCall.Name)
	}
	// With a failed ffprobe duration defaults to 0, so seek = 0*0.1 = 0.000
	if !argsContain(ffmpegCall.Args, "0.000") {
		t.Errorf("expected seek '0.000' in ffmpeg args, got: %v", ffmpegCall.Args)
	}
}

// --- trackingExec: ordered-response mock ---

type runResult struct {
	out []byte
	err error
}

type trackingExec struct {
	runs  []runResult
	calls []MockCall
	idx   int
}

func (te *trackingExec) Run(name string, args ...string) ([]byte, error) {
	te.calls = append(te.calls, MockCall{Name: name, Args: args})
	if te.idx < len(te.runs) {
		r := te.runs[te.idx]
		te.idx++
		return r.out, r.err
	}
	return nil, nil
}
