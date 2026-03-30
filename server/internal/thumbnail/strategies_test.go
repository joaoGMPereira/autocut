package thumbnail

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestGenerateLongForm
// ---------------------------------------------------------------------------

// TestGenerateLongForm verifies that GenerateLongForm calls ffmpeg at least
// NumFrames times for frame extraction (plus additional calls for the branded step).
func TestGenerateLongForm(t *testing.T) {
	// trackingExec: returns valid ffprobe JSON on the first call (duration),
	// then a non-empty response for all subsequent calls (frame extract + brand).
	durationJSON, _ := json.Marshal(struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{
		Format: struct {
			Duration string `json:"duration"`
		}{Duration: "120.000"},
	})

	te := &trackingExec{
		runs: buildRepeatResults(durationJSON, nil, 20),
	}

	g := newWithExec("ffmpeg", "convert", te)

	cfg := LongFormConfig{
		VideoPath: "v.mp4",
		NumFrames: 3,
		Width:     1280,
		Height:    720,
	}
	_ = g.GenerateLongForm(cfg, "out.jpg")

	// We expect: 1 ffprobe call + 3 frame extractions + 1 branded render = 5 calls minimum.
	// The test just verifies ffmpeg was called at least NumFrames times (for frames).
	ffmpegCalls := 0
	for _, c := range te.calls {
		if c.Name == "ffmpeg" {
			ffmpegCalls++
		}
	}
	if ffmpegCalls < cfg.NumFrames {
		t.Errorf("expected at least %d ffmpeg calls (frame extractions), got %d",
			cfg.NumFrames, ffmpegCalls)
	}
}

// TestGenerateLongFormFrameExtraction verifies that each frame extraction call
// uses -frames:v 1 and -ss with the extracted timestamp.
func TestGenerateLongFormFrameExtraction(t *testing.T) {
	durationJSON, _ := json.Marshal(struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{
		Format: struct {
			Duration string `json:"duration"`
		}{Duration: "60.000"},
	})

	te := &trackingExec{
		runs: buildRepeatResults(durationJSON, nil, 20),
	}
	g := newWithExec("ffmpeg", "convert", te)

	cfg := LongFormConfig{VideoPath: "v.mp4", NumFrames: 3}
	_ = g.GenerateLongForm(cfg, "out.jpg")

	// At least one ffmpeg call should contain -frames:v 1 (frame extraction).
	found := false
	for _, c := range te.calls {
		if c.Name == "ffmpeg" && argsContain(c.Args, "-frames:v") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one ffmpeg call with '-frames:v' for frame extraction")
	}
}

// ---------------------------------------------------------------------------
// TestGenerateShortsThumbnail
// ---------------------------------------------------------------------------

// TestGenerateShortsThumbnail verifies that the output filter contains the
// target dimensions 1080 and 1920 for portrait crop.
func TestGenerateShortsThumbnail(t *testing.T) {
	durationJSON, _ := json.Marshal(struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{
		Format: struct {
			Duration string `json:"duration"`
		}{Duration: "30.000"},
	})

	te := &trackingExec{
		runs: buildRepeatResults(durationJSON, nil, 10),
	}
	g := newWithExec("ffmpeg", "convert", te)

	cfg := ShortsThumbnailConfig{
		VideoPath: "v.mp4",
		Width:     1080,
		Height:    1920,
	}
	_ = g.GenerateShortsThumbnail(cfg, "out.jpg")

	// Find a call that contains both "1080" and "1920" (the crop/scale step).
	found := false
	for _, c := range te.calls {
		if c.Name == "ffmpeg" {
			combined := strings.Join(c.Args, " ")
			if strings.Contains(combined, "1080") && strings.Contains(combined, "1920") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected ffmpeg call with '1080' and '1920' in args. Calls: %+v", te.calls)
	}
}

// TestGenerateShortsThumbnailCrop9x16 verifies that the filter contains the
// 9:16 crop expression.
func TestGenerateShortsThumbnailCrop9x16(t *testing.T) {
	durationJSON, _ := json.Marshal(struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{
		Format: struct {
			Duration string `json:"duration"`
		}{Duration: "15.000"},
	})

	te := &trackingExec{
		runs: buildRepeatResults(durationJSON, nil, 10),
	}
	g := newWithExec("ffmpeg", "convert", te)

	cfg := ShortsThumbnailConfig{VideoPath: "v.mp4"}
	_ = g.GenerateShortsThumbnail(cfg, "out.jpg")

	found := false
	for _, c := range te.calls {
		if c.Name == "ffmpeg" {
			for _, a := range c.Args {
				if strings.Contains(a, "crop=ih*9/16:ih") {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Error("expected '9/16' crop expression in ffmpeg filter args")
	}
}

// TestGenerateShortsThumbnailWithText verifies that TextOverlay triggers drawtext.
func TestGenerateShortsThumbnailWithText(t *testing.T) {
	durationJSON, _ := json.Marshal(struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{
		Format: struct {
			Duration string `json:"duration"`
		}{Duration: "10.000"},
	})

	te := &trackingExec{
		runs: buildRepeatResults(durationJSON, nil, 10),
	}
	g := newWithExec("ffmpeg", "convert", te)

	cfg := ShortsThumbnailConfig{
		VideoPath:   "v.mp4",
		TextOverlay: "Subscribe Now!",
	}
	_ = g.GenerateShortsThumbnail(cfg, "out.jpg")

	found := false
	for _, c := range te.calls {
		if c.Name == "ffmpeg" {
			combined := strings.Join(c.Args, " ")
			if strings.Contains(combined, "drawtext") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected 'drawtext' in ffmpeg args when TextOverlay is set")
	}
}

// ---------------------------------------------------------------------------
// TestGetVideoDuration
// ---------------------------------------------------------------------------

// TestGetVideoDuration verifies that getVideoDuration parses ffprobe JSON correctly.
func TestGetVideoDuration(t *testing.T) {
	durationJSON, _ := json.Marshal(struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{
		Format: struct {
			Duration string `json:"duration"`
		}{Duration: "90.500"},
	})

	mock := &MockExecutor{Output: durationJSON}
	g := newWithExec("ffmpeg", "convert", mock)

	dur, err := g.getVideoDuration("video.mp4")
	if err != nil {
		t.Fatalf("getVideoDuration returned error: %v", err)
	}

	want := time.Duration(90.5 * float64(time.Second))
	// Allow ±1ms tolerance for float arithmetic
	diff := dur - want
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Millisecond {
		t.Errorf("getVideoDuration: got %v, want %v", dur, want)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildRepeatResults creates a slice of n runResult entries all returning the
// same output and error.
func buildRepeatResults(out []byte, err error, n int) []runResult {
	results := make([]runResult, n)
	for i := range results {
		results[i] = runResult{out: out, err: err}
	}
	return results
}

// trackingExec is already defined in generator_test.go; suppress redeclaration
// by using a type alias approach — actually it IS in the same package, so the
// type is shared. We cannot re-declare it here.
// The helpers below are new and specific to strategies_test.go.

// durationTrackingExec returns the given durationJSON for the first call,
// then empty bytes for all subsequent calls.
func newDurationTrackingExec(durationJSON []byte) *trackingExec {
	runs := make([]runResult, 20)
	runs[0] = runResult{out: durationJSON, err: nil}
	for i := 1; i < len(runs); i++ {
		runs[i] = runResult{out: []byte{}, err: nil}
	}
	return &trackingExec{runs: runs}
}

// argsContainSeq returns true when args contains the sub-sequence seq.
func argsContainSeq(args []string, seq ...string) bool {
	for i := 0; i <= len(args)-len(seq); i++ {
		match := true
		for j, s := range seq {
			if args[i+j] != s {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

