package ai

import (
	"fmt"
	"strings"
)

// engagementKeywordsPT contains high-engagement words for Brazilian Portuguese.
// Kotlin ref: ShortsOllamaDelegate.engagementKeywordsPT
var engagementKeywordsPT = []string{
	"incrível", "impossível", "olha", "nossa", "absurdo",
	"nunca", "sempre", "melhor", "pior", "secreto", "verdade",
	"chocante", "revelação", "urgente", "exclusivo", "segredo",
}

// engagementKeywordsEN contains high-engagement words for English.
// Kotlin ref: ShortsOllamaDelegate.engagementKeywordsEN
var engagementKeywordsEN = []string{
	"incredible", "impossible", "look", "wow", "amazing",
	"never", "always", "best", "worst", "secret", "truth",
	"shocking", "revealed", "urgent", "exclusive",
}

// engagementKeywordsES contains high-engagement words for Spanish.
var engagementKeywordsES = []string{
	"increíble", "imposible", "mira", "vaya", "asombroso",
	"nunca", "siempre", "mejor", "peor", "secreto", "verdad",
	"impactante", "revelado", "urgente", "exclusivo",
}

// keywordsForLang returns the keyword list for the given BCP-47 language tag.
// Falls back to Portuguese for unknown/empty values.
func keywordsForLang(lang string) []string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "en", "en-us", "en-gb":
		return engagementKeywordsEN
	case "es", "es-es", "es-mx", "es-ar":
		return engagementKeywordsES
	default:
		return engagementKeywordsPT
	}
}

// ScoreEngagement computes a composite engagement score for a short segment.
//
// Scoring algorithm (additive, max 100):
//
//  1. Confidence component (weight 0.40):
//     highlightConfidence [0,1] * 0.40 * 100 → up to 40 points
//
//  2. Speech-rate component (weight 0.30):
//     words/sec from transcript segments, peak at 3.0 w/s (clamp 0..6).
//     Normalised to [0,1] → * 0.30 * 100 → up to 30 points
//
//  3. Keyword component (max 25 points):
//     Each matching engagement keyword = +5 points, capped at 25.
//
//  4. Duration component (±10 points):
//     15 ≤ dur ≤ 45 s → +10 | > 60 or < 15 → -10 | 45 < dur ≤ 60 → 0
//
// Final score is clamped to [0, 100].
//
// Kotlin ref: ShortsOllamaDelegate.computeEngagementScore
func ScoreEngagement(
	highlightConfidence float64,
	segments []TranscriptSegment,
	durationSec float64,
	language string,
) EngagementScore {
	var score float64
	var factors []string

	// --- 1. Confidence component ---
	confidencePoints := highlightConfidence * 0.40 * 100
	score += confidencePoints
	if confidencePoints > 0 {
		factors = append(factors, fmt.Sprintf("highlight_confidence:%.2f", highlightConfidence))
	}

	// --- 2. Speech-rate component ---
	totalWords := 0
	for _, seg := range segments {
		totalWords += countWords(seg.GetText())
	}
	var speechRate float64
	if durationSec > 0 {
		speechRate = float64(totalWords) / durationSec
	}
	// Clamp to [0, 6], normalize against peak 3.0 w/s.
	clampedRate := speechRate
	if clampedRate > 6.0 {
		clampedRate = 6.0
	}
	// Normalise: 0..3 → 0..1 (ramp up), 3..6 → 1..0 (ramp down = too fast).
	var rateNorm float64
	if clampedRate <= 3.0 {
		rateNorm = clampedRate / 3.0
	} else {
		rateNorm = (6.0 - clampedRate) / 3.0
	}
	ratePoints := rateNorm * 0.30 * 100
	score += ratePoints
	if speechRate > 0 {
		factors = append(factors, fmt.Sprintf("speech_rate:%.1fw/s", speechRate))
	}

	// --- 3. Keyword component ---
	keywords := keywordsForLang(language)
	allText := joinSegmentText(segments)
	lower := strings.ToLower(allText)
	kwCount := 0
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			kwCount++
		}
	}
	kwPoints := float64(kwCount) * 5.0
	if kwPoints > 25.0 {
		kwPoints = 25.0
	}
	score += kwPoints
	if kwCount > 0 {
		factors = append(factors, fmt.Sprintf("keywords:%d", kwCount))
	}

	// --- 4. Duration component ---
	var durPoints float64
	var durLabel string
	switch {
	case durationSec >= 15 && durationSec <= 45:
		durPoints = 10
		durLabel = "duration:ideal"
	case durationSec > 60 || durationSec < 15:
		durPoints = -10
		durLabel = "duration:penalty"
	default: // 45 < dur <= 60
		durPoints = 0
		durLabel = "duration:ok"
	}
	score += durPoints
	factors = append(factors, durLabel)

	// --- Clamp ---
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return EngagementScore{
		Score:   score,
		Factors: factors,
	}
}

// countWords returns the number of whitespace-separated words in s.
func countWords(s string) int {
	return len(strings.Fields(s))
}

// joinSegmentText concatenates all segment texts with a space separator.
func joinSegmentText(segments []TranscriptSegment) string {
	parts := make([]string, 0, len(segments))
	for _, seg := range segments {
		if t := strings.TrimSpace(seg.GetText()); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}
