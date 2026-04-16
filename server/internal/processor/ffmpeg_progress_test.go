package processor

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
)

// TestRunFFmpegProgressUpdates verifies that onProgress fires multiple times
// during a real FFmpeg encode (not just once at the end).
// Requires ffmpeg on PATH; skipped otherwise.
func TestRunFFmpegProgressUpdates(t *testing.T) {
	if !FFmpegAvailable() {
		t.Skip("ffmpeg not on PATH")
	}

	outPath := fmt.Sprintf("%s/ffmpeg_progress_test_%d.mp4", os.TempDir(), os.Getpid())
	defer os.Remove(outPath)

	// Generate + encode a 10s 1080p video with a slow-ish preset so FFmpeg
	// has time to emit multiple progress lines.
	args := []string{
		"-y",
		"-f", "lavfi", "-i", "testsrc=duration=10:size=1920x1080:rate=30",
		"-c:v", "libx264", "-preset", "medium",
		outPath,
	}

	var mu sync.Mutex
	var updates []float64

	err := RunFFmpeg(context.Background(), args, 10.0, func(pct float64) {
		mu.Lock()
		updates = append(updates, pct)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("RunFFmpeg failed: %v", err)
	}

	mu.Lock()
	count := len(updates)
	mu.Unlock()

	t.Logf("received %d progress updates: %v", count, updates)

	if count < 2 {
		t.Errorf("expected multiple progress callbacks, got %d — scanner likely not splitting on \\r", count)
	}

	// Verify updates are monotonically non-decreasing
	for i := 1; i < count; i++ {
		if updates[i] < updates[i-1] {
			t.Errorf("progress went backwards: update[%d]=%.1f > update[%d]=%.1f", i-1, updates[i-1], i, updates[i])
		}
	}
}
