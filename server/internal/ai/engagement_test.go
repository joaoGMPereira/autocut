package ai

import (
	"testing"
)

// mockSeg is a test implementation of TranscriptSegment.
type mockSeg struct {
	start, end float64
	text       string
}

func (m mockSeg) GetStart() float64 { return m.start }
func (m mockSeg) GetEnd() float64   { return m.end }
func (m mockSeg) GetText() string   { return m.text }

// ---------------------------------------------------------------------------
// AC03 — Engagement scoring
// ---------------------------------------------------------------------------

// TestScoreEngagement_IdealShort verifies that a 30s short with high confidence
// and common PT keywords returns a score > 50.
func TestScoreEngagement_IdealShort(t *testing.T) {
	segs := []TranscriptSegment{
		mockSeg{0, 5, "incrível isso é impossível olha que absurdo"},
		mockSeg{5, 10, "nunca vi nada igual na minha vida"},
		mockSeg{10, 15, "nossa que coisa incrível mesmo"},
	}
	score := ScoreEngagement(0.80, segs, 30.0, "pt")
	if score.Score <= 50 {
		t.Errorf("expected score > 50 for ideal short, got %.2f (factors: %v)",
			score.Score, score.Factors)
	}
}

// TestScoreEngagement_ZeroDuration verifies scores don't panic on zero duration.
func TestScoreEngagement_ZeroDuration(t *testing.T) {
	score := ScoreEngagement(0.5, nil, 0, "pt")
	if score.Score < 0 || score.Score > 100 {
		t.Errorf("score out of [0,100]: %.2f", score.Score)
	}
}

// TestScoreEngagement_DurationPenalty verifies that very short (<15s) and very
// long (>60s) durations receive a penalty.
func TestScoreEngagement_DurationPenalty(t *testing.T) {
	// Score a short with high confidence but penalty duration.
	scoreShort := ScoreEngagement(0.9, nil, 10.0, "pt")
	scoreIdeal := ScoreEngagement(0.9, nil, 30.0, "pt")

	if scoreShort.Score >= scoreIdeal.Score {
		t.Errorf("duration penalty not applied: short=%.2f vs ideal=%.2f",
			scoreShort.Score, scoreIdeal.Score)
	}
}

// TestScoreEngagement_DurationIdeal verifies that 15-45s duration earns a bonus.
func TestScoreEngagement_DurationIdeal(t *testing.T) {
	// Ideal duration with no other signals
	scoreIdeal := ScoreEngagement(0, nil, 30.0, "pt")
	scoreNone := ScoreEngagement(0, nil, 55.0, "pt") // neutral zone

	if scoreIdeal.Score <= scoreNone.Score {
		t.Errorf("ideal duration should score higher: ideal=%.2f none=%.2f",
			scoreIdeal.Score, scoreNone.Score)
	}
}

// TestScoreEngagement_Clamped verifies that score is always in [0, 100].
func TestScoreEngagement_Clamped(t *testing.T) {
	cases := []struct {
		conf float64
		dur  float64
		lang string
	}{
		{1.0, 30.0, "pt"},
		{0.0, 5.0, "en"},
		{2.0, 100.0, "es"}, // out-of-spec inputs
	}
	for _, c := range cases {
		s := ScoreEngagement(c.conf, nil, c.dur, c.lang)
		if s.Score < 0 || s.Score > 100 {
			t.Errorf("score %.2f out of [0,100] for conf=%.1f dur=%.1f lang=%s",
				s.Score, c.conf, c.dur, c.lang)
		}
	}
}

// TestScoreEngagement_ENKeywords verifies that EN keywords are matched for lang="en".
func TestScoreEngagement_ENKeywords(t *testing.T) {
	segs := []TranscriptSegment{
		mockSeg{0, 5, "incredible secret revealed wow amazing"},
	}
	scoreEN := ScoreEngagement(0, segs, 30.0, "en")
	scoreNone := ScoreEngagement(0, nil, 30.0, "en")

	if scoreEN.Score <= scoreNone.Score {
		t.Errorf("EN keywords should increase score: with=%.2f without=%.2f",
			scoreEN.Score, scoreNone.Score)
	}
}

// TestScoreEngagement_Factors verifies that the factors slice is non-empty and
// contains at least a duration label.
func TestScoreEngagement_Factors(t *testing.T) {
	score := ScoreEngagement(0.8, nil, 30.0, "pt")
	if len(score.Factors) == 0 {
		t.Error("expected non-empty factors slice")
	}
	hasDuration := false
	for _, f := range score.Factors {
		if len(f) > 8 && f[:8] == "duration" {
			hasDuration = true
		}
	}
	if !hasDuration {
		t.Errorf("expected a duration factor, got: %v", score.Factors)
	}
}

// TestScoreEngagement_HighlightIDZero verifies HighlightID defaults to 0.
func TestScoreEngagement_HighlightIDZero(t *testing.T) {
	score := ScoreEngagement(0.5, nil, 30.0, "pt")
	if score.HighlightID != 0 {
		t.Errorf("expected HighlightID=0, got %d", score.HighlightID)
	}
}

// TestKeywordsForLang_Defaults verifies that unknown/empty language returns PT.
func TestKeywordsForLang_Defaults(t *testing.T) {
	if len(keywordsForLang("")) == 0 {
		t.Error("expected non-empty keyword list for empty lang")
	}
	if len(keywordsForLang("fr")) == 0 {
		t.Error("expected fallback PT keywords for unknown lang 'fr'")
	}
}
