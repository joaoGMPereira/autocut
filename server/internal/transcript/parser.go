package transcript

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// whisperSegmentJSON mirrors the whisper.cpp --output-json segment shape.
// Kotlin ref: TranscriptJsonParser.parseWhisperJson — reads "transcription"[].offsets.from/to ms.
// whisper.cpp native JSON uses "transcription" at the top level with millisecond offsets.
type whisperSegmentJSON struct {
	Offsets struct {
		From int `json:"from"` // milliseconds
		To   int `json:"to"`   // milliseconds
	} `json:"offsets"`
	Text string `json:"text"`
}

type whisperOutputJSON struct {
	Transcription []whisperSegmentJSON `json:"transcription"`
}

// ParseWhisperJSON parses the JSON emitted by whisper-cli with --output-json.
//
// Expected shape (whisper.cpp native format):
//
//	{"transcription": [{"offsets": {"from": 0, "to": 1500}, "text": "hello"}]}
//
// Returns error if the JSON is malformed or the "transcription" key is absent.
// An empty transcription array is valid and returns a Transcript with 0 segments.
func ParseWhisperJSON(data []byte) (*Transcript, error) {
	if len(data) == 0 {
		return nil, errors.New("ParseWhisperJSON: empty input")
	}

	var raw whisperOutputJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("ParseWhisperJSON: unmarshal: %w", err)
	}

	// Validate that "transcription" key was present (not just zero-value absent).
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err == nil {
		if _, ok := probe["transcription"]; !ok {
			return nil, errors.New("ParseWhisperJSON: missing \"transcription\" key")
		}
	}

	segments := make([]Segment, 0, len(raw.Transcription))
	var maxEnd time.Duration

	for _, s := range raw.Transcription {
		start := msToDuration(s.Offsets.From)
		end := msToDuration(s.Offsets.To)

		seg := Segment{
			Start:      start,
			End:        end,
			Text:       s.Text,
			Confidence: 0.9, // whisper CLI does not expose per-segment confidence
		}

		segments = append(segments, seg)

		if end > maxEnd {
			maxEnd = end
		}
	}

	lang := ""
	if lv, ok := probe["language"]; ok {
		_ = json.Unmarshal(lv, &lang)
	}

	return &Transcript{
		Segments: segments,
		Language: lang,
		Duration: maxEnd,
	}, nil
}

// msToDuration converts a millisecond integer to time.Duration.
func msToDuration(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
