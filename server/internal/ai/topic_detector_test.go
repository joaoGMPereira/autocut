package ai

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// MockSegment — implements TranscriptSegment for tests
// ---------------------------------------------------------------------------

// MockSegment is a test double for TranscriptSegment.
type MockSegment struct {
	start float64
	end   float64
	text  string
}

func (m MockSegment) GetStart() float64 { return m.start }
func (m MockSegment) GetEnd() float64   { return m.end }
func (m MockSegment) GetText() string   { return m.text }

// seg is a convenience constructor.
func seg(start, end float64, text string) MockSegment {
	return MockSegment{start: start, end: end, text: text}
}

// ---------------------------------------------------------------------------
// TestFilterMusic
// ---------------------------------------------------------------------------

// TestFilterMusic verifies that segments with very low chars-per-second ratio
// are classified as music/noise.
func TestFilterMusic(t *testing.T) {
	cfg := DetectorConfig{MusicThreshold: 0.3}
	d := NewDetector(nil, cfg) // client not needed for this method

	segments := []TranscriptSegment{
		seg(0, 30, "ok"), // 2 chars / 30s = 0.067 → music
		seg(30, 60, "Hoje vamos falar sobre como usar o Go para construir servidores web rápidos"), // dense → speech
		seg(60, 65, "a"),  // 1 char / 5s = 0.2 → music
		seg(65, 70, "Continuando nosso tutorial de Go backend com SQLite e NDJSON streaming"), // dense → speech
	}

	got := d.filterMusic(segments)

	// Indices 0 and 2 should be flagged.
	if len(got) != 2 {
		t.Fatalf("expected 2 music segments, got %d: %v", len(got), got)
	}
	if got[0] != 0 || got[1] != 2 {
		t.Errorf("expected [0, 2], got %v", got)
	}
}

// ---------------------------------------------------------------------------
// TestDetectExplicitTransitions
// ---------------------------------------------------------------------------

// TestDetectExplicitTransitions verifies that a segment containing a known
// transition keyword is returned.
func TestDetectExplicitTransitions(t *testing.T) {
	d := NewDetector(nil, DetectorConfig{})

	segments := []TranscriptSegment{
		seg(0, 10, "Bem-vindos ao canal, hoje vamos falar sobre Go"),
		seg(10, 20, "Primeiro vamos instalar as dependências"),
		seg(20, 30, "próximo tópico é sobre testes"),
		seg(30, 40, "vamos escrever um handler HTTP simples"),
	}

	got := d.detectExplicitTransitions(segments)

	// Segments 0 ("hoje vamos"), and 2 ("próximo tópico") should be detected.
	if len(got) < 2 {
		t.Fatalf("expected at least 2 explicit transitions, got %d: %v", len(got), got)
	}

	inResult := func(idx int) bool {
		for _, g := range got {
			if g == idx {
				return true
			}
		}
		return false
	}

	if !inResult(0) {
		t.Errorf("expected segment 0 ('hoje vamos') to be detected")
	}
	if !inResult(2) {
		t.Errorf("expected segment 2 ('próximo tópico') to be detected")
	}
}

// ---------------------------------------------------------------------------
// TestAnalyzeEmpty
// ---------------------------------------------------------------------------

// TestAnalyzeEmpty verifies that Analyze returns a zero-value AnalysisResult
// (not an error) when the input is empty.
func TestAnalyzeEmpty(t *testing.T) {
	d := NewDetector(nil, DetectorConfig{})

	result, err := d.Analyze(context.Background(), nil)
	if err != nil {
		t.Fatalf("Analyze(nil) returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil AnalysisResult")
	}
	if len(result.Topics) != 0 {
		t.Errorf("expected 0 topics, got %d", len(result.Topics))
	}
	if len(result.Highlights) != 0 {
		t.Errorf("expected 0 highlights, got %d", len(result.Highlights))
	}
	if len(result.MusicSegmentIndices) != 0 {
		t.Errorf("expected 0 music indices, got %d", len(result.MusicSegmentIndices))
	}
}

// ---------------------------------------------------------------------------
// TestMergeBoundaries
// ---------------------------------------------------------------------------

// TestMergeBoundaries is a unit-level check for the deduplication logic.
func TestMergeBoundaries(t *testing.T) {
	a := []int{2, 7, 15}
	b := []int{3, 7, 20} // 3 is within ±1 of 2; 7 is duplicate; 20 is new

	got := mergeBoundaries(a, b)

	// Expected: [2, 7, 15, 20]  (3 deduplicated with 2, 7 deduplicated)
	want := []int{2, 7, 15, 20}
	if len(got) != len(want) {
		t.Fatalf("mergeBoundaries: want %v got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mergeBoundaries[%d]: want %d got %d", i, want[i], got[i])
		}
	}
}

// ---------------------------------------------------------------------------
// TestExtractJSONArray
// ---------------------------------------------------------------------------

func TestExtractJSONArray(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`[1,2,3]`, `[1,2,3]`},
		{"Here are results: [4,5]\nDone.", `[4,5]`},
		{"```json\n[\"a\",\"b\"]\n```", `["a","b"]`},
		{"no array here", "no array here"},
	}
	for _, c := range cases {
		got := extractJSONArray(c.in)
		if got != c.want {
			t.Errorf("extractJSONArray(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
