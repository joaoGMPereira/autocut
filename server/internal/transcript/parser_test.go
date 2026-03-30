package transcript

import (
	"testing"
	"time"
)

func TestParseBasic(t *testing.T) {
	data := []byte(`{
		"segments": [
			{"start": 0.0,  "end": 1.5,  "text": "hello world"},
			{"start": 2.0,  "end": 4.25, "text": "second segment"}
		]
	}`)

	tr, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tr.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(tr.Segments))
	}

	s0 := tr.Segments[0]
	if s0.Start != 0 {
		t.Errorf("seg[0].Start: want 0, got %v", s0.Start)
	}
	if s0.End != 1500*time.Millisecond {
		t.Errorf("seg[0].End: want 1.5s, got %v", s0.End)
	}
	if s0.Text != "hello world" {
		t.Errorf("seg[0].Text: want %q, got %q", "hello world", s0.Text)
	}

	s1 := tr.Segments[1]
	if s1.Start != 2*time.Second {
		t.Errorf("seg[1].Start: want 2s, got %v", s1.Start)
	}
	wantEnd := time.Duration(4.25 * float64(time.Second))
	if s1.End != wantEnd {
		t.Errorf("seg[1].End: want %v, got %v", wantEnd, s1.End)
	}

	// Duration should equal the last segment's End.
	if tr.Duration != wantEnd {
		t.Errorf("Duration: want %v, got %v", wantEnd, tr.Duration)
	}
}

func TestParseEmpty(t *testing.T) {
	data := []byte(`{"segments": []}`)

	tr, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("unexpected error on empty segments: %v", err)
	}
	if len(tr.Segments) != 0 {
		t.Errorf("expected 0 segments, got %d", len(tr.Segments))
	}
	if tr.Duration != 0 {
		t.Errorf("expected zero Duration, got %v", tr.Duration)
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
	}{
		{"empty_bytes", []byte{}},
		{"bad_json", []byte(`{not json}`)},
		{"no_segments_key", []byte(`{"transcription": []}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := ParseWhisperJSON(tc.input)
			if err == nil {
				t.Errorf("expected error for %q, got nil (transcript=%+v)", tc.name, tr)
			}
		})
	}
}

func TestParseWithWords(t *testing.T) {
	// Ensures that a JSON with a "words" array parses without panic.
	data := []byte(`{
		"segments": [
			{
				"start": 0.0,
				"end": 2.0,
				"text": "hello there",
				"words": [
					{"word": "hello", "start": 0.0, "end": 0.8},
					{"word": "there", "start": 0.9, "end": 2.0}
				]
			}
		]
	}`)

	tr, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tr.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(tr.Segments))
	}
	if tr.Segments[0].Text != "hello there" {
		t.Errorf("unexpected text: %q", tr.Segments[0].Text)
	}
}
