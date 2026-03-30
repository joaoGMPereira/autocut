package transcript

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// whisperSegmentJSON mirrors the whisper.cpp --output-json-full segment shape.
// Kotlin ref: TranscriptJsonParser.parseWhisperJson — Kotlin reads "transcription"[].offsets.from/to ms;
// whisper --output-json (not full) uses "segments"[].start/end as float64 seconds.
// We target the simpler --output-json format used by whisper CLI wrappers.
type whisperSegmentJSON struct {
	Start float64            `json:"start"`
	End   float64            `json:"end"`
	Text  string             `json:"text"`
	Words []whisperWordJSON  `json:"words"`
}

type whisperWordJSON struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type whisperOutputJSON struct {
	Segments []whisperSegmentJSON `json:"segments"`
}

// ParseWhisperJSON parses the JSON emitted by whisper CLI with --output-json.
//
// Expected shape:
//
//	{"segments": [{"start": 0.0, "end": 1.5, "text": "hello", "words": [...]}]}
//
// Returns error if the JSON is malformed or the "segments" key is absent.
// An empty segments array is valid and returns a Transcript with 0 segments.
func ParseWhisperJSON(data []byte) (*Transcript, error) {
	if len(data) == 0 {
		return nil, errors.New("ParseWhisperJSON: empty input")
	}

	var raw whisperOutputJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("ParseWhisperJSON: unmarshal: %w", err)
	}

	// Validate that "segments" key was present (not just zero-value absent).
	// We use a secondary check by re-unmarshalling into a map.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err == nil {
		if _, ok := probe["segments"]; !ok {
			return nil, errors.New("ParseWhisperJSON: missing \"segments\" key")
		}
	}

	segments := make([]Segment, 0, len(raw.Segments))
	var maxEnd time.Duration

	for _, s := range raw.Segments {
		start := floatSecsToDuration(s.Start)
		end := floatSecsToDuration(s.End)

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

// floatSecsToDuration converts a float64 seconds value to time.Duration.
func floatSecsToDuration(secs float64) time.Duration {
	return time.Duration(secs * float64(time.Second))
}
