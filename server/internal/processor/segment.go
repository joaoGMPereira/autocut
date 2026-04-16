package processor

import (
	"context"
	"fmt"
	"math"
)

const (
	defaultSegmentSec = 600.0 // 10 minutes
	minSegmentSec     = 300.0 // 5 minutes — below this, clamp to default
)

// Segment is a {StartSec, EndSec} pair produced by ComputeSegments.
// It corresponds 1:1 to a PipelineClip row.
type Segment struct {
	StartSec float64
	EndSec   float64
}

// ComputeSegments splits totalSec into segments of segmentSec with smart redistribution:
// if the final remainder would be < 50% of segmentSec, it is merged into the penultimate segment.
// Returns nil if totalSec <= 0.
// If segmentSec <= 0 or < minSegmentSec, it is clamped to defaultSegmentSec.
func ComputeSegments(totalSec, segmentSec float64) []Segment {
	if totalSec <= 0 {
		return nil
	}
	if segmentSec < minSegmentSec {
		segmentSec = defaultSegmentSec
	}
	if totalSec <= segmentSec {
		return []Segment{{StartSec: 0, EndSec: totalSec}}
	}

	n := int(math.Floor(totalSec / segmentSec))
	remainder := totalSec - float64(n)*segmentSec

	segments := make([]Segment, 0, n+1)
	for i := 0; i < n-1; i++ {
		segments = append(segments, Segment{
			StartSec: float64(i) * segmentSec,
			EndSec:   float64(i+1) * segmentSec,
		})
	}

	lastStart := float64(n-1) * segmentSec
	if remainder >= 0.5*segmentSec {
		// Significant remainder — emit as a separate final segment.
		segments = append(segments, Segment{StartSec: lastStart, EndSec: lastStart + segmentSec})
		segments = append(segments, Segment{StartSec: lastStart + segmentSec, EndSec: totalSec})
	} else {
		// Tiny remainder — merge into the last segment.
		segments = append(segments, Segment{StartSec: lastStart, EndSec: totalSec})
	}

	return segments
}

// CutSegment cuts videoPath from startSec for durationSec seconds using FFmpeg stream copy.
// The output directory must already exist. Returns error if FFmpeg exits non-zero.
// onProgress is called with 0–100 as ffmpeg processes the segment; pass nil to discard.
func CutSegment(ctx context.Context, videoPath string, startSec, durationSec float64, outPath string, onProgress func(float64)) error {
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", startSec),
		"-i", videoPath,
		"-t", fmt.Sprintf("%.3f", durationSec),
		"-c", "copy",
		outPath,
	}
	return RunFFmpeg(ctx, args, durationSec, onProgress)
}
