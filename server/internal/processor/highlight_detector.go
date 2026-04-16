package processor

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// HighlightSegment is the input unit for highlight detection.
// StartSec and EndSec are seconds from the beginning of the video.
type HighlightSegment struct {
	StartSec float64
	EndSec   float64
	Text     string
}

// defaultKeywords is the default set of high-value keywords used for scoring.
var defaultKeywords = []string{
	"important", "key", "critical", "main", "highlight",
	"focus", "note", "warning", "tip", "trick", "secret",
}

// DetectHighlights scores transcript segments using a three-factor local heuristic
// and returns highlight candidates with score >= 0.2.
// Segments are scored by pause gap (weight 0.5), keyword density (weight 0.3),
// and segment length (weight 0.2). All scoring is deterministic and in-memory.
// onProgress receives values in [0, 100] every 10 segments and at the end.
func DetectHighlights(segments []HighlightSegment, onProgress func(pct float64)) []database.PipelineHighlight {
	total := len(segments)
	if total == 0 {
		slog.Info("DetectHighlights: no segments to score")
		if onProgress != nil {
			onProgress(100)
		}
		return nil
	}

	slog.Info("DetectHighlights: scoring segments", "total", total)
	now := time.Now().UnixMilli()
	var results []database.PipelineHighlight
	var allScored []database.PipelineHighlight // kept for fallback ranking

	for i, seg := range segments {
		// --- Pause gap score (weight 0.5) ---
		var pauseScore float64
		var gapSec float64
		if i > 0 {
			gapSec = seg.StartSec - segments[i-1].EndSec
			if gapSec > 2.0 {
				pauseScore = math.Min(1.0, gapSec/10.0)
			}
		}

		// --- Keyword density score (weight 0.3) ---
		lower := strings.ToLower(seg.Text)
		matchCount := 0
		for _, kw := range defaultKeywords {
			if strings.Contains(lower, kw) {
				matchCount++
			}
		}
		keywordScore := math.Min(1.0, float64(matchCount)*0.2)

		// --- Segment length score (weight 0.2) ---
		segLen := seg.EndSec - seg.StartSec
		lengthScore := math.Min(1.0, segLen/30.0)

		// --- Combined score ---
		score := pauseScore*0.5 + keywordScore*0.3 + lengthScore*0.2
		if score > 1.0 {
			score = 1.0
		}

		// --- Padding ---
		adjStart := math.Max(0, seg.StartSec-1.5)
		adjEnd := seg.EndSec + 1.5

		// --- Human-readable reason ---
		reason := buildReason(pauseScore, keywordScore, lengthScore, gapSec)

		h := database.PipelineHighlight{
			StartSec:    seg.StartSec,
			EndSec:      seg.EndSec,
			AdjStartSec: adjStart,
			AdjEndSec:   adjEnd,
			Score:       score,
			Text:        seg.Text,
			Reason:      reason,
			IsSelected:  false,
			CreatedAt:   now,
		}
		allScored = append(allScored, h)

		if score >= 0.2 {
			results = append(results, h)
		}

		// Emit progress every 10 segments and at the final segment.
		if (i+1)%10 == 0 || i == total-1 {
			if onProgress != nil {
				onProgress(float64(i+1) / float64(total) * 100)
			}
		}
	}

	// Fallback: if no segment passed the threshold (e.g. continuous speech with no
	// pauses or matching keywords), take the top-10 by relative score so the review
	// screen always has candidates to display.
	if len(results) == 0 && len(allScored) > 0 {
		sort.Slice(allScored, func(i, j int) bool { return allScored[i].Score > allScored[j].Score })
		topN := 10
		if topN > len(allScored) {
			topN = len(allScored)
		}
		for _, h := range allScored[:topN] {
			h.Reason = "top relative score"
			results = append(results, h)
		}
		slog.Info("DetectHighlights: fallback — top relative scores", "picked", len(results))
	}

	slog.Info("DetectHighlights: done", "total_segments", total, "highlights_found", len(results))
	return results
}

// buildReason constructs a human-readable explanation for why a segment was selected.
func buildReason(pauseScore, keywordScore, lengthScore, gapSec float64) string {
	var parts []string
	if pauseScore > 0 {
		parts = append(parts, fmt.Sprintf("pause gap (%.0fs silence)", gapSec))
	}
	if keywordScore > 0 {
		parts = append(parts, "keyword match")
	}
	if lengthScore >= 0.5 {
		parts = append(parts, "long speech segment")
	}
	if len(parts) == 0 {
		return "threshold score"
	}
	return strings.Join(parts, " + ")
}
