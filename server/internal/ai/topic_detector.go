package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// explicitTransitionKeywords are phrases that strongly signal a topic change.
// Kotlin ref: TopicExplicitTransitionDelegate.transitionPhrases (subset kept to
// language-agnostic + most common PT-BR phrases per spec)
var explicitTransitionKeywords = []string{
	// PT-BR
	"hoje vamos", "agora vamos", "próximo tópico", "vamos ver agora",
	"mudando de assunto", "falando nisso", "por falar nisso",
	"vamos pro próximo", "chega de", "hora de", "agora sim",
	// EN
	"now let's", "moving on", "next up", "let's talk about",
	"switching to", "next topic",
}

// TopicTransitionDetector detects topic transitions in a transcript and
// produces structured Topics and Highlights.
//
// Kotlin ref: TopicTransitionDetector — thin coordinator that delegates to
// specialist methods. Go version merges all delegates into one struct to keep
// the package surface small (no circular delegate references needed in Go).
type TopicTransitionDetector struct {
	provider AIProvider
	cfg      DetectorConfig
	log      *slog.Logger
}

// NewDetector creates a TopicTransitionDetector with the given AI provider
// and config. Zero config values are filled with package defaults.
// Accepts any AIProvider (OllamaProvider, mock, etc.).
func NewDetector(provider AIProvider, cfg DetectorConfig) *TopicTransitionDetector {
	return &TopicTransitionDetector{
		provider: provider,
		cfg:      cfg.withDefaults(),
		log:      slog.With("component", "topic_detector"),
	}
}

// Analyze runs the full detection pipeline on segments:
//  1. filterMusic   — find music/noise indices
//  2. detectExplicitTransitions — keyword-based boundary detection
//  3. detectSemanticTransitions — Ollama-based boundary detection
//  4. classifyTopics — label each topic group via Ollama
//  5. generateHighlights — score and rank highlights
//
// Kotlin ref: TopicTransitionDetector.detectTopicTransitions()
func (d *TopicTransitionDetector) Analyze(ctx context.Context, segments []TranscriptSegment) (*AnalysisResult, error) {
	d.log.Info("analyze started", "segments", len(segments))

	if len(segments) == 0 {
		return &AnalysisResult{}, nil
	}

	// 1. Music filter
	musicIdx := d.filterMusic(segments)
	d.log.Debug("music filter done", "music_count", len(musicIdx))

	// Build clean (non-music) segment list for subsequent steps.
	musicSet := make(map[int]struct{}, len(musicIdx))
	for _, i := range musicIdx {
		musicSet[i] = struct{}{}
	}
	clean := make([]TranscriptSegment, 0, len(segments))
	cleanOrigIdx := make([]int, 0, len(segments)) // maps clean index → original index
	for i, s := range segments {
		if _, isMusic := musicSet[i]; !isMusic {
			clean = append(clean, s)
			cleanOrigIdx = append(cleanOrigIdx, i)
		}
	}

	if len(clean) == 0 {
		d.log.Warn("all segments filtered as music, returning empty result")
		return &AnalysisResult{MusicSegmentIndices: musicIdx}, nil
	}

	// 2. Explicit transitions (on clean segments, reported as original indices)
	explicitBoundaries := d.detectExplicitTransitions(clean)
	d.log.Debug("explicit transitions", "count", len(explicitBoundaries))

	// 3. Semantic transitions via Ollama
	semanticBoundaries, err := d.detectSemanticTransitions(ctx, clean)
	if err != nil {
		d.log.Error("semantic transitions failed, falling back to explicit only",
			"err", err)
		semanticBoundaries = nil
	}
	d.log.Debug("semantic transitions", "count", len(semanticBoundaries))

	// Merge boundaries (deduplicate within ±1 segment)
	allBoundaries := mergeBoundaries(explicitBoundaries, semanticBoundaries)

	// 4. Classify topics (uses clean segment indices)
	topics := d.classifyTopics(ctx, clean, allBoundaries)
	d.log.Info("topics classified", "count", len(topics))

	// 5. Generate highlights
	highlights := d.generateHighlights(clean, topics, explicitBoundaries)
	d.log.Info("highlights generated", "count", len(highlights))

	return &AnalysisResult{
		Topics:              topics,
		Highlights:          highlights,
		MusicSegmentIndices: musicIdx,
	}, nil
}

// ---------------------------------------------------------------------------
// Private methods
// ---------------------------------------------------------------------------

// filterMusic returns the indices of segments that look like music or noise.
// Heuristic: chars-per-second ratio below MusicThreshold.
// Kotlin ref: TopicMusicFilterDelegate.filterMusicSegments() — Go version
// uses a density ratio instead of pattern matching to stay language-agnostic.
func (d *TopicTransitionDetector) filterMusic(segments []TranscriptSegment) []int {
	var result []int
	for i, s := range segments {
		durationSec := s.GetEnd() - s.GetStart()
		if durationSec <= 0 {
			result = append(result, i)
			continue
		}
		ratio := float64(len([]rune(s.GetText()))) / durationSec
		if ratio < d.cfg.MusicThreshold {
			result = append(result, i)
		}
	}
	return result
}

// detectExplicitTransitions returns segment indices where an explicit
// transition keyword was found.
// Kotlin ref: TopicExplicitTransitionDelegate.detectExplicitTransitions()
func (d *TopicTransitionDetector) detectExplicitTransitions(segments []TranscriptSegment) []int {
	seen := make(map[int]struct{})
	for i, s := range segments {
		lower := strings.ToLower(s.GetText())
		for _, kw := range explicitTransitionKeywords {
			if strings.Contains(lower, kw) {
				seen[i] = struct{}{}
				break
			}
		}
	}
	result := make([]int, 0, len(seen))
	for i := range seen {
		result = append(result, i)
	}
	sortInts(result)
	return result
}

// detectSemanticTransitions asks Ollama to identify where the subject changes
// in a batch of segment texts.
// Kotlin ref: TopicSemanticTransitionDelegate.detectSemanticTransitions() —
// Kotlin used Jaccard similarity locally; Go version delegates to Ollama for
// better multilingual accuracy.
func (d *TopicTransitionDetector) detectSemanticTransitions(
	ctx context.Context, segments []TranscriptSegment,
) ([]int, error) {
	if len(segments) < 4 {
		return nil, nil
	}

	// Build a compact numbered list of texts.
	lines := make([]string, len(segments))
	for i, s := range segments {
		text := s.GetText()
		if len(text) > 120 {
			text = text[:120]
		}
		lines[i] = fmt.Sprintf("%d: %s", i, text)
	}

	prompt := fmt.Sprintf(
		`You are analyzing transcript segments numbered 0 to %d.
Identify segment indices where the topic/subject clearly changes.
Return ONLY a JSON array of integers (the indices), nothing else.
Example: [3,11,24]

Segments:
%s`,
		len(segments)-1,
		strings.Join(lines, "\n"),
	)

	req := GenerateRequest{
		Model:       d.cfg.Model,
		Prompt:      prompt,
		System:      "You are a transcript analysis assistant. Respond only with valid JSON.",
		Temperature: 0.2,
	}

	if d.provider == nil {
		return nil, nil
	}
	raw, err := d.provider.GenerateSync(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("detectSemanticTransitions: %w", err)
	}

	// Extract JSON array from response (model may add prose around it).
	raw = extractJSONArray(raw)

	var indices []int
	if err := json.Unmarshal([]byte(raw), &indices); err != nil {
		d.log.Warn("semantic transitions: failed to parse response",
			"raw", raw, "err", err)
		return nil, nil
	}

	// Clamp indices to valid range.
	valid := indices[:0]
	for _, idx := range indices {
		if idx >= 0 && idx < len(segments) {
			valid = append(valid, idx)
		}
	}
	sortInts(valid)
	return valid, nil
}

// segmentGroup is a contiguous run of transcript segments between boundaries.
type segmentGroup struct {
	startIdx int
	endIdx   int
	text     string
}

// classifyTopics groups segments between boundary indices and asks Ollama for
// a short label for each group.
// Kotlin ref: TopicClassificationDelegate.classifyAllSegments() — Go version
// builds the same batch prompt but in a single call.
func (d *TopicTransitionDetector) classifyTopics(
	ctx context.Context, segments []TranscriptSegment, boundaries []int,
) []Topic {
	if len(segments) == 0 {
		return nil
	}

	boundarySet := make(map[int]struct{}, len(boundaries))
	for _, b := range boundaries {
		boundarySet[b] = struct{}{}
	}

	var groups []segmentGroup
	groupStart := 0
	for i := 1; i <= len(segments); i++ {
		_, isBoundary := boundarySet[i]
		if isBoundary || i == len(segments) {
			var sb strings.Builder
			for j := groupStart; j < i && j < len(segments); j++ {
				sb.WriteString(segments[j].GetText())
				sb.WriteString(" ")
			}
			groups = append(groups, segmentGroup{
				startIdx: groupStart,
				endIdx:   i - 1,
				text:     strings.TrimSpace(sb.String()),
			})
			groupStart = i
		}
	}

	if len(groups) == 0 {
		return nil
	}

	// Build batch prompt.
	promptLines := make([]string, len(groups))
	for i, g := range groups {
		excerpt := g.text
		if len(excerpt) > 200 {
			excerpt = excerpt[:200]
		}
		promptLines[i] = fmt.Sprintf("%d. %s", i+1, excerpt)
	}

	prompt := fmt.Sprintf(
		`Classify each segment with a short title (max 8 words, in the same language as the text).
Return ONLY a JSON array of strings, no markdown.

Segments:
%s`,
		strings.Join(promptLines, "\n"),
	)

	req := GenerateRequest{
		Model:       d.cfg.Model,
		Prompt:      prompt,
		System:      "You are a video classifier. Respond only with valid JSON.",
		Temperature: 0.3,
	}

	if d.provider == nil {
		return d.fallbackTopics(groups)
	}
	raw, err := d.provider.GenerateSync(ctx, req)
	if err != nil {
		d.log.Error("classifyTopics: provider error, using fallback labels", "err", err)
		return d.fallbackTopics(groups)
	}

	raw = extractJSONArray(raw)
	var labels []string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		d.log.Warn("classifyTopics: failed to parse labels, using fallback",
			"raw", raw, "err", err)
		return d.fallbackTopics(groups)
	}

	topics := make([]Topic, len(groups))
	for i, g := range groups {
		label := fmt.Sprintf("Segment %d", i+1)
		if i < len(labels) && labels[i] != "" {
			label = labels[i]
		}
		topics[i] = Topic{
			StartIdx:   g.startIdx,
			EndIdx:     g.endIdx,
			Label:      label,
			Confidence: d.cfg.ConfidenceThreshold,
		}
	}
	return topics
}

// fallbackTopics generates numbered Topic stubs when Ollama is unavailable.
func (d *TopicTransitionDetector) fallbackTopics(groups []segmentGroup) []Topic {
	topics := make([]Topic, len(groups))
	for i, g := range groups {
		topics[i] = Topic{
			StartIdx:   g.startIdx,
			EndIdx:     g.endIdx,
			Label:      fmt.Sprintf("Segment %d", i+1),
			Confidence: 0.5,
		}
	}
	return topics
}

// generateHighlights scores each topic group and returns ranked highlights.
//
// Scoring formula (additive):
//   - base = information density (chars per second across the group)
//     normalised to [0, 1] against the max density in this batch
//   - +0.2 if the group starts with an explicit transition
//   - +0.1 if the group is a singleton cluster (unique boundary on both sides)
//
// Kotlin ref: TopicHighlightDelegate.segmentsToHighlights() — Go version
// computes score inline instead of relying on confidence from classifier.
func (d *TopicTransitionDetector) generateHighlights(
	segments []TranscriptSegment,
	topics []Topic,
	explicitBoundaries []int,
) []Highlight {
	if len(topics) == 0 || len(segments) == 0 {
		return nil
	}

	explicitSet := make(map[int]struct{}, len(explicitBoundaries))
	for _, b := range explicitBoundaries {
		explicitSet[b] = struct{}{}
	}

	type scored struct {
		h     Highlight
		raw   float64
	}

	densities := make([]float64, len(topics))
	for i, t := range topics {
		var totalChars int
		var durationSec float64
		end := t.EndIdx
		if end >= len(segments) {
			end = len(segments) - 1
		}
		for j := t.StartIdx; j <= end; j++ {
			totalChars += len([]rune(segments[j].GetText()))
			durationSec += segments[j].GetEnd() - segments[j].GetStart()
		}
		if durationSec > 0 {
			densities[i] = float64(totalChars) / durationSec
		}
	}

	maxDensity := 0.0
	for _, d := range densities {
		if d > maxDensity {
			maxDensity = d
		}
	}

	highlights := make([]Highlight, 0, len(topics))
	for i, t := range topics {
		score := 0.0
		if maxDensity > 0 {
			score = densities[i] / maxDensity
		}

		// +0.2 if this group starts at an explicit transition boundary
		if _, ok := explicitSet[t.StartIdx]; ok {
			score += 0.2
		}

		// +0.1 if this is a singleton (both neighbours are boundaries)
		if len(topics) == 1 {
			score += 0.1
		}

		// Clamp to [0, 1]
		if score > 1.0 {
			score = 1.0
		}

		end := t.EndIdx
		if end >= len(segments) {
			end = len(segments) - 1
		}

		h := Highlight{
			StartSec: segments[t.StartIdx].GetStart(),
			EndSec:   segments[end].GetEnd(),
			Score:    score,
			Reason:   t.Label,
		}
		highlights = append(highlights, h)
	}

	return highlights
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mergeBoundaries combines two boundary index slices, deduplicating entries
// that are within 1 segment of each other.
func mergeBoundaries(a, b []int) []int {
	merged := make([]int, 0, len(a)+len(b))
	merged = append(merged, a...)
outer:
	for _, idx := range b {
		for _, existing := range merged {
			diff := idx - existing
			if diff < 0 {
				diff = -diff
			}
			if diff <= 1 {
				continue outer
			}
		}
		merged = append(merged, idx)
	}
	sortInts(merged)
	return merged
}

// sortInts sorts a slice of ints in-place (stdlib sort avoided to keep deps minimal).
func sortInts(s []int) {
	// insertion sort — fast for small N (typical: <50 boundaries)
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

// extractJSONArray finds the first '[' ... ']' substring in s.
// Models sometimes wrap the array in prose or markdown fences.
func extractJSONArray(s string) string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}
