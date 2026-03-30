package transcript

import "time"

// Segment represents a single transcribed segment with timing and text.
// Kotlin ref: TranscriptSegment (start/end as float64 seconds → time.Duration here)
type Segment struct {
	Start      time.Duration `json:"start"`
	End        time.Duration `json:"end"`
	Text       string        `json:"text"`
	Confidence float64       `json:"confidence"`
}

// Transcript is the result of a full transcription run.
// Kotlin ref: Transcript (videoPath dropped — not relevant to the Go layer)
type Transcript struct {
	Segments []Segment     `json:"segments"`
	Language string        `json:"language"`
	Duration time.Duration `json:"duration"`
}

// WhisperConfig holds configuration for the WhisperTranscriber.
// Defaults: BinPath="whisper", ChunkDuration=25min.
type WhisperConfig struct {
	// BinPath is the name or absolute path of the whisper binary (default: "whisper").
	BinPath string `yaml:"bin_path"`
	// ModelPath is the path to the .bin model file.
	ModelPath string `yaml:"model_path"`
	// Language passed to --language (e.g. "pt", "en", "auto").
	Language string `yaml:"language"`
	// ChunkDuration is the max segment length before chunking kicks in (default: 25min).
	ChunkDuration time.Duration `yaml:"chunk_duration"`
}

// defaults fills zero values with sensible defaults.
func (c *WhisperConfig) defaults() {
	if c.BinPath == "" {
		c.BinPath = "whisper"
	}
	if c.ChunkDuration == 0 {
		c.ChunkDuration = 25 * time.Minute
	}
	if c.Language == "" {
		c.Language = "auto"
	}
}
