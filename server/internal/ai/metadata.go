package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// MetadataGenerator orchestrates AI-powered metadata generation for pipeline clips.
type MetadataGenerator struct {
	cli           *ClaudeCLI
	runRepo       *database.PipelineRunRepo
	clipRepo      *database.PipelineClipRepo
	highlightRepo *database.PipelineHighlightRepo
	channelRepo   *database.ChannelConfigRepo
	settingRepo   *database.AppSettingRepo
	hub           *hub.SSEHub
	log           *slog.Logger
}

// NewMetadataGenerator constructs a MetadataGenerator. cli may be nil if Claude is unavailable.
func NewMetadataGenerator(
	cli *ClaudeCLI,
	runRepo *database.PipelineRunRepo,
	clipRepo *database.PipelineClipRepo,
	highlightRepo *database.PipelineHighlightRepo,
	channelRepo *database.ChannelConfigRepo,
	settingRepo *database.AppSettingRepo,
	h *hub.SSEHub,
) *MetadataGenerator {
	return &MetadataGenerator{
		cli:           cli,
		runRepo:       runRepo,
		clipRepo:      clipRepo,
		highlightRepo: highlightRepo,
		channelRepo:   channelRepo,
		settingRepo:   settingRepo,
		hub:           h,
		log:           slog.With("component", "metadata_generator"),
	}
}

// Available returns true if the Claude CLI is ready.
func (g *MetadataGenerator) Available() bool {
	return g.cli != nil
}

// GenerateBatch generates metadata for all clips in a run using Claude CLI.
// Publishes SSE progress events during generation.
func (g *MetadataGenerator) GenerateBatch(ctx context.Context, runID int64) ([]ClipMetadata, error) {
	if g.cli == nil {
		return nil, fmt.Errorf("claude CLI not available")
	}

	jobKey := fmt.Sprintf("%d", runID)

	// 1. Load run
	run, err := g.runRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load run %d: %w", runID, err)
	}

	// 2. Load clips
	clips, err := g.clipRepo.ListByRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load clips for run %d: %w", runID, err)
	}
	if len(clips) == 0 {
		return nil, fmt.Errorf("no clips found for run %d", runID)
	}

	// 3. Publish "generating" status
	g.publishProgress(jobKey, runID, "generating", 10, fmt.Sprintf("Preparing prompt for %d clips...", len(clips)), len(clips))

	// 4. Load channel config (optional)
	var channelTags, channelCategory string
	if run.ChannelID.Valid {
		cfg, cfgErr := g.channelRepo.GetByChannelID(ctx, run.ChannelID.Int64)
		if cfgErr == nil {
			channelTags = cfg.DefaultTags
			if cfg.DefaultCategoryID > 0 {
				channelCategory = fmt.Sprintf("%d", cfg.DefaultCategoryID)
			}
		}
	}

	// 5. Load selected highlights for context
	highlights, _ := g.highlightRepo.ListSelectedByRun(ctx, runID)

	// 6. Read transcript (if exists)
	var transcript string
	if run.TranscriptPath != "" {
		data, readErr := os.ReadFile(run.TranscriptPath)
		if readErr == nil {
			transcript = string(data)
			// Truncate to ~8000 chars to stay within context limits
			if len(transcript) > 8000 {
				transcript = transcript[:8000] + "\n[... truncated]"
			}
		}
	}

	// 7. Read model setting
	model := "haiku"
	if m, _ := g.settingRepo.Get(ctx, "metadata_ai_model"); m != "" {
		model = m
	}

	// 8. Build prompts
	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(run, clips, highlights, channelTags, channelCategory, transcript)

	g.publishProgress(jobKey, runID, "streaming", 30, "Claude is generating metadata...", len(clips))

	// 9. Call Claude
	resultText, err := g.cli.GenerateStream(ctx, GenerateRequest{
		Prompt:       userPrompt,
		SystemPrompt: systemPrompt,
		Model:        model,
	}, func(delta string) {
		g.publishProgress(jobKey, runID, "streaming", 60, "Generating...", len(clips))
	})
	if err != nil {
		g.publishProgress(jobKey, runID, "error", 0, fmt.Sprintf("Generation failed: %s", err), len(clips))
		return nil, fmt.Errorf("claude generate: %w", err)
	}

	g.publishProgress(jobKey, runID, "streaming", 80, "Parsing response...", len(clips))

	// 10. Parse JSON response
	metadata, err := parseMetadataResponse(resultText, clips)
	if err != nil {
		g.publishProgress(jobKey, runID, "error", 0, fmt.Sprintf("Failed to parse response: %s", err), len(clips))
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 11. Persist to database
	for _, m := range metadata {
		tags := strings.Join(m.Tags, ",")
		if updateErr := g.clipRepo.UpdateMetadata(ctx, m.ClipID, m.Title, m.Description, tags, m.ThumbnailText); updateErr != nil {
			g.log.Warn("failed to update clip metadata", "clip_id", m.ClipID, "err", updateErr)
		}
	}

	g.publishProgress(jobKey, runID, "done", 100, "Metadata generated successfully", len(clips))
	return metadata, nil
}

// GenerateSingle generates metadata for a single clip.
func (g *MetadataGenerator) GenerateSingle(ctx context.Context, runID, clipID int64) (*ClipMetadata, error) {
	if g.cli == nil {
		return nil, fmt.Errorf("claude CLI not available")
	}

	run, err := g.runRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load run %d: %w", runID, err)
	}

	clips, err := g.clipRepo.ListByRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load clips: %w", err)
	}

	// Find the target clip
	var targetClip *database.PipelineClip
	for i := range clips {
		if clips[i].ID == clipID {
			targetClip = &clips[i]
			break
		}
	}
	if targetClip == nil {
		return nil, fmt.Errorf("clip %d not found in run %d", clipID, runID)
	}

	// Load highlight for this clip
	var highlightText, highlightReason string
	var highlightScore float64
	if targetClip.HighlightID.Valid {
		highlights, _ := g.highlightRepo.ListSelectedByRun(ctx, runID)
		for _, h := range highlights {
			if h.ID == targetClip.HighlightID.Int64 {
				highlightText = h.Text
				highlightReason = h.Reason
				highlightScore = h.Score
				break
			}
		}
	}

	// Read transcript segment for this clip
	var transcriptSegment string
	if run.TranscriptPath != "" {
		data, readErr := os.ReadFile(run.TranscriptPath)
		if readErr == nil {
			transcriptSegment = extractTranscriptSegment(string(data), targetClip.StartSec, targetClip.EndSec)
		}
	}

	// Channel context
	var channelTags string
	if run.ChannelID.Valid {
		cfg, cfgErr := g.channelRepo.GetByChannelID(ctx, run.ChannelID.Int64)
		if cfgErr == nil {
			channelTags = cfg.DefaultTags
		}
	}

	model := "haiku"
	if m, _ := g.settingRepo.Get(ctx, "metadata_ai_model"); m != "" {
		model = m
	}

	systemPrompt := buildSystemPrompt()
	userPrompt := fmt.Sprintf(`Generate YouTube metadata for 1 clip from this video.

## Source Video
- Title: %s
- URL: %s

## Channel Tags
%s

## Clip (%.1fs - %.1fs, %.1fs duration)
Highlight: %s
Reason: %s
Score: %.1f
Transcript: %s

Return a JSON array with exactly 1 object:
[{"thumbnail_text":"SHORT HOOK","title":"...","description":"...","tags":["..."],"category_id":22}]`,
		run.VideoTitle, run.URL, channelTags,
		targetClip.StartSec, targetClip.EndSec, targetClip.DurationSec,
		highlightText, highlightReason, highlightScore, transcriptSegment)

	resultText, err := g.cli.Generate(ctx, GenerateRequest{
		Prompt:       userPrompt,
		SystemPrompt: systemPrompt,
		Model:        model,
	})
	if err != nil {
		return nil, fmt.Errorf("claude generate: %w", err)
	}

	metadata, err := parseMetadataResponse(resultText, []database.PipelineClip{*targetClip})
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(metadata) == 0 {
		return nil, fmt.Errorf("no metadata generated")
	}

	// Persist
	m := metadata[0]
	tags := strings.Join(m.Tags, ",")
	if updateErr := g.clipRepo.UpdateMetadata(ctx, m.ClipID, m.Title, m.Description, tags, m.ThumbnailText); updateErr != nil {
		g.log.Warn("failed to update clip metadata", "clip_id", m.ClipID, "err", updateErr)
	}

	return &m, nil
}

// publishProgress emits an SSE metadata_progress event.
func (g *MetadataGenerator) publishProgress(jobKey string, runID int64, status string, percent int, message string, clipCount int) {
	g.hub.Publish(jobKey, hub.SSEEvent{
		Type: "metadata_progress",
		Data: map[string]interface{}{
			"run_id":     runID,
			"status":     status,
			"percent":    percent,
			"message":    message,
			"clip_count": clipCount,
		},
	})
}

// buildSystemPrompt returns the system prompt for metadata generation.
func buildSystemPrompt() string {
	return `You are a YouTube SEO metadata expert. Generate optimized metadata for video clips.

Rules:
- thumbnail_text: MAX 3 words, ALL CAPS, impactful hook for thumbnail overlay (e.g. "INSANE PLAY", "NO WAY", "EPIC FAIL")
- title: Under 100 characters, attention-grabbing, include relevant keywords
- description: 200-500 characters, brief summary + keywords + hashtags
- tags: 5-15 relevant keywords/phrases, each tag max 30 characters
- category_id: YouTube category (20=Gaming, 22=People & Blogs, 24=Entertainment, 27=Education, 28=Science & Tech)
- Return ONLY valid JSON, no markdown fences, no explanation`
}

// buildUserPrompt constructs the user prompt with all available context.
func buildUserPrompt(
	run *database.PipelineRun,
	clips []database.PipelineClip,
	highlights []database.PipelineHighlight,
	channelTags, channelCategory, transcript string,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate YouTube metadata for %d clips from this video.\n\n", len(clips)))

	// Source video
	sb.WriteString("## Source Video\n")
	sb.WriteString(fmt.Sprintf("- Title: %s\n", run.VideoTitle))
	sb.WriteString(fmt.Sprintf("- URL: %s\n", run.URL))
	sb.WriteString(fmt.Sprintf("- Duration: %ds\n\n", run.DurationSec))

	// Channel context
	if channelTags != "" || channelCategory != "" {
		sb.WriteString("## Channel Context\n")
		if channelTags != "" {
			sb.WriteString(fmt.Sprintf("- Default tags: %s\n", channelTags))
		}
		if channelCategory != "" {
			sb.WriteString(fmt.Sprintf("- Category: %s\n", channelCategory))
		}
		sb.WriteString("\n")
	}

	// Clips
	sb.WriteString("## Clips\n")
	highlightMap := make(map[int64]database.PipelineHighlight)
	for _, h := range highlights {
		highlightMap[h.ID] = h
	}

	for i, clip := range clips {
		sb.WriteString(fmt.Sprintf("### Clip %d (%.1fs - %.1fs, %.1fs)\n", i+1, clip.StartSec, clip.EndSec, clip.DurationSec))
		if clip.HighlightID.Valid {
			if h, ok := highlightMap[clip.HighlightID.Int64]; ok {
				sb.WriteString(fmt.Sprintf("Highlight: %s\n", h.Text))
				sb.WriteString(fmt.Sprintf("Reason: %s\n", h.Reason))
				sb.WriteString(fmt.Sprintf("Score: %.1f\n", h.Score))
			}
		}
		// Add transcript segment for this clip
		if transcript != "" {
			segment := extractTranscriptSegment(transcript, clip.StartSec, clip.EndSec)
			if segment != "" {
				sb.WriteString(fmt.Sprintf("Transcript: %s\n", segment))
			}
		}
		sb.WriteString("\n")
	}

	// Output format
	sb.WriteString("## Output\n")
	sb.WriteString(fmt.Sprintf("JSON array with exactly %d objects:\n", len(clips)))
	sb.WriteString(`[{"thumbnail_text":"SHORT HOOK","title":"...","description":"...","tags":["..."],"category_id":22}]`)
	sb.WriteString("\n")

	return sb.String()
}

// extractTranscriptSegment returns the portion of transcript text that falls within the given time range.
// Uses a proportional slice of the full transcript as a heuristic, since transcript formats vary
// (SRT, VTT, plain text, etc.).
func extractTranscriptSegment(transcript string, startSec, endSec float64) string {
	totalLen := len(transcript)
	if totalLen == 0 {
		return ""
	}

	// Estimate proportion based on clip position relative to total transcript length.
	// Use endSec*2 as a rough proxy for total video duration when it's larger than totalLen.
	denom := float64(totalLen)
	if endSec*2 > denom {
		denom = endSec * 2
	}

	startRatio := startSec / denom
	endRatio := endSec / denom

	startIdx := int(startRatio * float64(totalLen))
	endIdx := int(endRatio * float64(totalLen))

	// Clamp
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > totalLen {
		endIdx = totalLen
	}
	if startIdx >= endIdx {
		return ""
	}

	// Expand to word boundaries
	for startIdx > 0 && transcript[startIdx] != ' ' && transcript[startIdx] != '\n' {
		startIdx--
	}
	for endIdx < totalLen && transcript[endIdx] != ' ' && transcript[endIdx] != '\n' {
		endIdx++
	}

	segment := strings.TrimSpace(transcript[startIdx:endIdx])

	// Limit to 2000 chars per clip
	if len(segment) > 2000 {
		segment = segment[:2000] + "..."
	}

	return segment
}

// parseMetadataResponse extracts the JSON array from Claude's response and maps to clip IDs.
func parseMetadataResponse(text string, clips []database.PipelineClip) ([]ClipMetadata, error) {
	// Strip markdown code fences if present
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
	}
	text = strings.TrimSpace(text)

	// Find JSON array in the text
	startIdx := strings.Index(text, "[")
	endIdx := strings.LastIndex(text, "]")
	if startIdx < 0 || endIdx < 0 || endIdx <= startIdx {
		return nil, fmt.Errorf("no JSON array found in response: %.200s", text)
	}
	jsonText := text[startIdx : endIdx+1]

	var raw []ClipMetadata
	if err := json.Unmarshal([]byte(jsonText), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w (text: %.200s)", err, jsonText)
	}

	// Map by array index to clip IDs
	result := make([]ClipMetadata, 0, len(clips))
	for i, clip := range clips {
		if i >= len(raw) {
			break
		}
		m := raw[i]
		m.ClipID = clip.ID
		result = append(result, m)
	}

	return result, nil
}
