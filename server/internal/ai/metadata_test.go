package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// makeAnalytics is a helper to construct a ChannelAnalytics with JSON-encoded fields.
func makeAnalytics(words []WordCount, tags []TagCount, titles []string) *database.ChannelAnalytics {
	topWords, _ := json.Marshal(words)
	topTags, _ := json.Marshal(tags)
	successTitles, _ := json.Marshal(titles)
	return &database.ChannelAnalytics{
		ChannelID:     1,
		VideoCount:    50,
		AvgViews:      10000,
		TopTitleWords: string(topWords),
		TopTags:       string(topTags),
		SuccessTitles: string(successTitles),
		RawTitles:     "[]",
	}
}

// makeRun returns a minimal PipelineRun for testing.
func makeRun() *database.PipelineRun {
	return &database.PipelineRun{
		ID:          1,
		URL:         "https://www.youtube.com/watch?v=test",
		VideoTitle:  "Test Video",
		DurationSec: 3600,
	}
}

// makeClips returns a slice with one clip for testing.
func makeClips() []database.PipelineClip {
	return []database.PipelineClip{
		{ID: 1, StartSec: 10, EndSec: 70, DurationSec: 60},
	}
}

func TestBuildUserPrompt_WithAnalytics(t *testing.T) {
	words := []WordCount{
		{Word: "golang", Count: 5},
		{Word: "tutorial", Count: 3},
	}
	tags := []TagCount{
		{Tag: "programação", Count: 10},
		{Tag: "golang", Count: 7},
	}
	titles := []string{
		"Como aprender Go em 30 dias",
		"Erros fatais no Go que todo dev comete",
	}
	analytics := makeAnalytics(words, tags, titles)

	prompt := buildUserPrompt(makeRun(), makeClips(), nil, "", "", "", analytics, "Canal Teste")

	if !strings.Contains(prompt, "## Contexto do Canal: Canal Teste") {
		t.Errorf("expected '## Contexto do Canal: Canal Teste' in prompt, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Vídeos analisados: 50") {
		t.Errorf("expected 'Vídeos analisados: 50' in prompt")
	}
	if !strings.Contains(prompt, "Média de views: 10000") {
		t.Errorf("expected 'Média de views: 10000' in prompt")
	}
	if !strings.Contains(prompt, `"golang"`) {
		t.Errorf("expected word 'golang' in top title words section")
	}
	if !strings.Contains(prompt, "Como aprender Go em 30 dias") {
		t.Errorf("expected success title in prompt")
	}
	if !strings.Contains(prompt, "programação") {
		t.Errorf("expected tag 'programação' in prompt")
	}
}

func TestBuildUserPrompt_WithoutAnalytics(t *testing.T) {
	prompt := buildUserPrompt(makeRun(), makeClips(), nil, "", "", "", nil, "")

	if strings.Contains(prompt, "## Contexto do Canal") {
		t.Errorf("expected NO '## Contexto do Canal' section when analytics is nil, but found it in:\n%s", prompt)
	}
}

func TestBuildUserPrompt_AnalyticsTopWordsCappedAt10(t *testing.T) {
	// Build 20 words — only the first 10 should appear in the prompt.
	words := make([]WordCount, 20)
	for i := 0; i < 20; i++ {
		words[i] = WordCount{Word: generateWord(i), Count: 20 - i}
	}
	analytics := makeAnalytics(words, nil, nil)

	prompt := buildUserPrompt(makeRun(), makeClips(), nil, "", "", "", analytics, "")

	// Count how many of our generated words appear
	count := 0
	for i := 0; i < 20; i++ {
		if strings.Contains(prompt, generateWord(i)) {
			count++
		}
	}

	if count > 10 {
		t.Errorf("expected at most 10 top words in prompt, but found %d words from the 20-word list", count)
	}
	if count < 1 {
		t.Errorf("expected at least 1 top word in prompt, got 0")
	}
}

// generateWord creates a unique word for index i that won't collide with prose.
func generateWord(i int) string {
	return strings.Repeat("x", i+3) + "word"
}
