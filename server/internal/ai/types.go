package ai

// GenerateRequest is the payload sent to Ollama's /api/generate endpoint.
// Kotlin ref: OllamaClient.generate() parameters
type GenerateRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	System      string  `json:"system,omitempty"`
	Temperature float64 `json:"temperature"`
	Stream      bool    `json:"stream"`
}

// Topic represents a classified content segment identified between transitions.
// Kotlin ref: TopicTransitionDetector.TopicSegment (flattened — no keywords/description)
type Topic struct {
	StartIdx   int     `json:"start_idx"`
	EndIdx     int     `json:"end_idx"`
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

// Highlight is a scored time range recommended for cutting.
// Kotlin ref: Highlight (timestamp+duration collapsed into StartSec/EndSec)
type Highlight struct {
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
}

// AnalysisResult is the combined output of a full Analyze() call.
// Kotlin ref: TopicTransitionDetector.TopicAnalysisResult
type AnalysisResult struct {
	Topics              []Topic   `json:"topics"`
	Highlights          []Highlight `json:"highlights"`
	MusicSegmentIndices []int     `json:"music_segment_indices"`
}

// DetectorConfig holds tunable parameters for TopicTransitionDetector.
// Zero values are replaced with package defaults via withDefaults().
// Kotlin ref: AIConfig (OllamaURL/thresholds extracted from constructor args)
type DetectorConfig struct {
	// Model is the Ollama model name (e.g. "qwen2.5:7b").
	Model string `yaml:"model"`
	// OllamaURL is the base URL of the Ollama HTTP server.
	// Default: "http://localhost:11434"
	OllamaURL string `yaml:"ollama_url"`
	// MusicThreshold is the chars-per-second ratio below which a segment is
	// considered music/noise. Default: 0.3
	MusicThreshold float64 `yaml:"music_threshold"`
	// ConfidenceThreshold is the minimum confidence to accept a topic label.
	// Default: 0.7
	ConfidenceThreshold float64 `yaml:"confidence_threshold"`
}

// withDefaults returns a copy of cfg with zero values replaced by package defaults.
func (cfg DetectorConfig) withDefaults() DetectorConfig {
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}
	if cfg.MusicThreshold == 0 {
		cfg.MusicThreshold = 0.3
	}
	if cfg.ConfidenceThreshold == 0 {
		cfg.ConfidenceThreshold = 0.7
	}
	if cfg.Model == "" {
		cfg.Model = "qwen2.5:7b"
	}
	return cfg
}

// TranscriptSegment is a read-only view of a transcribed segment.
// Defined as an interface to avoid import cycles with the transcript package —
// transcript.Segment satisfies this interface automatically.
// Kotlin ref: TranscriptSegment (Kotlin uses data class; Go uses interface for DI)
type TranscriptSegment interface {
	GetStart() float64 // segment start in seconds
	GetEnd() float64   // segment end in seconds
	GetText() string   // raw transcribed text
}
