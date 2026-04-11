package transcript

import (
	"testing"
	"time"
)

func TestParseBasic(t *testing.T) {
	// whisper.cpp native format: "transcription"[].offsets.{from, to} in milliseconds
	data := []byte(`{
		"transcription": [
			{"offsets": {"from": 0,    "to": 1500}, "text": "hello world"},
			{"offsets": {"from": 2000, "to": 4250}, "text": "second segment"}
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
	if s1.End != 4250*time.Millisecond {
		t.Errorf("seg[1].End: want 4.25s, got %v", s1.End)
	}

	// Duration should equal the last segment's End.
	if tr.Duration != 4250*time.Millisecond {
		t.Errorf("Duration: want 4.25s, got %v", tr.Duration)
	}
}

func TestParseEmpty(t *testing.T) {
	data := []byte(`{"transcription": []}`)

	tr, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("unexpected error on empty transcription: %v", err)
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
		// Old "segments" format is no longer valid — parser expects "transcription".
		{"no_transcription_key", []byte(`{"segments": []}`)},
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

func TestParseLanguage(t *testing.T) {
	// whisper.cpp may include a top-level "language" field.
	data := []byte(`{
		"transcription": [
			{"offsets": {"from": 0, "to": 1000}, "text": "olá mundo"}
		],
		"language": "pt"
	}`)

	tr, err := ParseWhisperJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Language != "pt" {
		t.Errorf("Language: want %q, got %q", "pt", tr.Language)
	}
	if len(tr.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(tr.Segments))
	}
}
