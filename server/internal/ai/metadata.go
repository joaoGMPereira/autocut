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
	cli            *ClaudeCLI
	runRepo        *database.PipelineRunRepo
	clipRepo       *database.PipelineClipRepo
	highlightRepo  *database.PipelineHighlightRepo
	channelRepo    *database.ChannelConfigRepo
	channelBaseRepo *database.ChannelRepo
	settingRepo    *database.AppSettingRepo
	analyticsRepo  *database.ChannelAnalyticsRepo
	hub            *hub.SSEHub
	analyzer       *ChannelAnalyzer
	log            *slog.Logger
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
	analyticsRepo *database.ChannelAnalyticsRepo, // may be nil
	analyzer *ChannelAnalyzer,                    // may be nil
) *MetadataGenerator {
	return &MetadataGenerator{
		cli:           cli,
		runRepo:       runRepo,
		clipRepo:      clipRepo,
		highlightRepo: highlightRepo,
		channelRepo:   channelRepo,
		settingRepo:   settingRepo,
		hub:           h,
		analyticsRepo: analyticsRepo,
		analyzer:      analyzer,
		log:           slog.With("component", "metadata_generator"),
	}
}

// SetChannelBaseRepo sets the ChannelRepo used for looking up channel URL/title.
// Called during wiring (T4) — optional, gracefully degrades when nil.
func (g *MetadataGenerator) SetChannelBaseRepo(r *database.ChannelRepo) {
	g.channelBaseRepo = r
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
	g.publishProgress(jobKey, runID, "generating", 10, fmt.Sprintf("Preparando prompt para %d clips...", len(clips)), len(clips))

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

	// 5. Load channel analytics from cache (transparent to caller)
	var analytics *database.ChannelAnalytics
	var channelName string
	if run.ChannelID.Valid && g.analyticsRepo != nil {
		a, aErr := g.analyticsRepo.Get(ctx, run.ChannelID.Int64, AnalyticsTTLSec)
		if aErr != nil {
			g.log.Warn("failed to load channel analytics", "err", aErr)
		} else {
			analytics = a
		}
	}
	if run.ChannelID.Valid && g.channelBaseRepo != nil {
		ch, _ := g.channelBaseRepo.GetByID(ctx, run.ChannelID.Int64)
		if ch != nil {
			channelName = ch.ChannelTitle
		}
	}

	// 6. Load selected highlights for context
	highlights, _ := g.highlightRepo.ListSelectedByRun(ctx, runID)

	// 7. Read transcript (if exists)
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

	// 8. Read model setting
	model := "haiku"
	if m, _ := g.settingRepo.Get(ctx, "metadata_ai_model"); m != "" {
		model = m
	}

	// 9. Build prompts
	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(run, clips, highlights, channelTags, channelCategory, transcript, analytics, channelName)

	g.publishProgress(jobKey, runID, "streaming", 30, "Claude está gerando metadados...", len(clips))

	// 10. Call Claude
	resultText, err := g.cli.GenerateStream(ctx, GenerateRequest{
		Prompt:       userPrompt,
		SystemPrompt: systemPrompt,
		Model:        model,
	}, func(delta string) {
		g.publishProgress(jobKey, runID, "streaming", 60, "Gerando...", len(clips))
	})
	if err != nil {
		g.publishProgress(jobKey, runID, "error", 0, fmt.Sprintf("Generation failed: %s", err), len(clips))
		return nil, fmt.Errorf("claude generate: %w", err)
	}

	g.publishProgress(jobKey, runID, "streaming", 80, "Processando resposta...", len(clips))

	// 11. Parse JSON response
	metadata, err := parseMetadataResponse(resultText, clips)
	if err != nil {
		g.publishProgress(jobKey, runID, "error", 0, fmt.Sprintf("Failed to parse response: %s", err), len(clips))
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 12. Persist to database
	for _, m := range metadata {
		tags := strings.Join(m.Tags, ",")
		if updateErr := g.clipRepo.UpdateMetadata(ctx, m.ClipID, m.Title, m.Description, tags, m.ThumbnailText); updateErr != nil {
			g.log.Warn("failed to update clip metadata", "clip_id", m.ClipID, "err", updateErr)
		}
	}

	g.publishProgress(jobKey, runID, "done", 100, "Metadados gerados com sucesso", len(clips))
	return metadata, nil
}

// PrefillBatch is designed to be called in a goroutine. It optionally fetches fresh
// channel analytics (cache-first) before delegating to GenerateBatch.
// All errors are published as SSE events — the method never returns an error.
func (g *MetadataGenerator) PrefillBatch(ctx context.Context, runID int64) {
	jobKey := fmt.Sprintf("%d", runID)

	// 1. Load run
	run, err := g.runRepo.GetByID(ctx, runID)
	if err != nil {
		g.publishProgress(jobKey, runID, "error", 0, fmt.Sprintf("Failed to load run: %s", err), 0)
		return
	}

	// 2. Optionally ensure fresh channel analytics
	if run.ChannelID.Valid && g.analyzer != nil && g.analyticsRepo != nil {
		channelIDInt := run.ChannelID.Int64

		// Check cache first
		cached, cacheErr := g.analyticsRepo.Get(ctx, channelIDInt, AnalyticsTTLSec)
		if cacheErr != nil {
			g.log.Warn("prefill: analytics cache lookup failed", "err", cacheErr)
		}

		if cached == nil {
			// Cache miss: fetch from YouTube.
			// Build channel URL — requires a real YouTube channel ID string from channelBaseRepo.
			// Never fall back to the DB integer PK: it produces an invalid URL that yt-dlp will 404.
			var channelURL string
			if g.channelBaseRepo != nil {
				ch, chErr := g.channelBaseRepo.GetByID(ctx, channelIDInt)
				if chErr == nil && ch != nil && ch.ChannelID != "" {
					channelURL = "https://www.youtube.com/channel/" + ch.ChannelID
				}
			}

			if channelURL != "" {
				g.publishProgress(jobKey, runID, "fetching_channel", 10, "Analisando histórico do canal...", 0)
				_, fetchErr := g.analyzer.FetchAndCache(ctx, channelIDInt, channelURL)
				if fetchErr != nil {
					// Graceful degradation: log and continue to generation.
					g.log.Warn("prefill: failed to fetch channel analytics", "err", fetchErr)
				}
			} else {
				g.log.Warn("prefill: no channel URL available, skipping analytics fetch")
			}
		}
	}

	// 3. Delegate to GenerateBatch (analytics loaded transparently inside).
	// GenerateBatch publishes its own "generating" + "done" events — don't double-emit here.
	_, genErr := g.GenerateBatch(ctx, runID)
	if genErr != nil {
		g.publishProgress(jobKey, runID, "error", 0, fmt.Sprintf("Generation failed: %s", genErr), 0)
		return
	}
	// done is already published inside GenerateBatch
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
	userPrompt := fmt.Sprintf(`Gere metadados YouTube para 1 clip deste vídeo.

## Vídeo Fonte
- Título: %s
- URL: %s

## Tags do Canal
%s

## Clip (%.1fs - %.1fs, %.1fs de duração)
Destaque: %s
Razão: %s
Score: %.1f
Transcrição: %s

Retorne um array JSON com exatamente 1 objeto:
[{"thumbnail_text":"TEXTO IMPACTO","title":"...","description":"...","tags":["..."],"category_id":22}]`,
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
	return `Você é um especialista em SEO de YouTube e criação de conteúdo viral para clips longos.

TÍTULOS (max 100 chars, CAIXA ALTA):
- Front-load: benefício/keyword principal nas 5 primeiras palavras
- Curiosity Gap: prometa resultado ou crie lacuna de curiosidade
- Especificidade: use números e resultados concretos
- Anti-genérico: evite títulos vagos; seja ultra-específico
- TÍTULOS AUTÔNOMOS: funcionem sozinhos, sem depender de sequência
- SEM NUMERAÇÃO: NUNCA use "Parte 1", "PT1" ou números (mata CTR)

TÉCNICAS DE HOOK (priorize):
1. Pergunta de Curiosidade: "Por que ninguém está falando sobre X?"
2. Pergunta de Resultado: "Como eu fiz X em apenas Y tempo?"
3. Pergunta de Erro: "Você ainda está cometendo este erro em X?"
4. Afirmação Contraintuitiva: "Pare de fazer X agora mesmo"
5. Comparação Direta: "X vs Y: Qual é realmente melhor?"
6. Lista Específica: "7 segredos para X que mudaram meu jogo"

DESCRIÇÕES:
- Above the Fold: primeiros 150 chars = gancho principal + keyword
- Corpo Semântico: ferramentas, pessoas e conceitos citados pelo NOME (Knowledge Graph)
- CTA simples no final
- Citabilidade: escreva para que LLMs possam citar como resposta

THUMBNAIL TEXT (max 38 chars, CAIXA ALTA):
- 1-3 palavras de alto impacto visual
- Anti-genérico: evite "INCRÍVEL", "LEGAL" — prefira "ERRO FATAL", "LUCRO REAL"
- Complemento: adiciona camada de mistério que a imagem sozinha não conta

TAGS: 5-8 tags semânticas de categorização (não keyword stuffing)
IDIOMA: 100% Português do Brasil
OUTPUT: JSON válido, sem markdown, sem explicações`
}

// buildUserPrompt constructs the user prompt with all available context.
// analytics is optional — when non-nil, a channel context section is injected before the clips.
func buildUserPrompt(
	run *database.PipelineRun,
	clips []database.PipelineClip,
	highlights []database.PipelineHighlight,
	channelTags, channelCategory, transcript string,
	analytics *database.ChannelAnalytics,
	channelName string,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Gere metadados YouTube para %d clips deste vídeo.\n\n", len(clips)))

	// Source video
	sb.WriteString("## Vídeo Fonte\n")
	sb.WriteString(fmt.Sprintf("- Título: %s\n", run.VideoTitle))
	sb.WriteString(fmt.Sprintf("- URL: %s\n", run.URL))
	sb.WriteString(fmt.Sprintf("- Duração: %ds\n\n", run.DurationSec))

	// Channel context from config
	if channelTags != "" || channelCategory != "" {
		sb.WriteString("## Contexto do Canal\n")
		if channelTags != "" {
			sb.WriteString(fmt.Sprintf("- Tags padrão: %s\n", channelTags))
		}
		if channelCategory != "" {
			sb.WriteString(fmt.Sprintf("- Categoria: %s\n", channelCategory))
		}
		sb.WriteString("\n")
	}

	// Analytics context section (injected before clips)
	if analytics != nil {
		name := channelName
		if name == "" {
			name = "Canal"
		}
		sb.WriteString(fmt.Sprintf("## Contexto do Canal: %s\n", name))
		sb.WriteString(fmt.Sprintf("- Vídeos analisados: %d | Média de views: %d\n", analytics.VideoCount, analytics.AvgViews))

		// Top title words (cap at 10)
		if analytics.TopTitleWords != "" && analytics.TopTitleWords != "[]" {
			var words []WordCount
			if err := json.Unmarshal([]byte(analytics.TopTitleWords), &words); err == nil && len(words) > 0 {
				if len(words) > 10 {
					words = words[:10]
				}
				sb.WriteString("\n### Palavras mais usadas nos títulos:\n")
				parts := make([]string, 0, len(words))
				for _, w := range words {
					parts = append(parts, fmt.Sprintf("%q (%dx)", w.Word, w.Count))
				}
				sb.WriteString("- " + strings.Join(parts, ", ") + "\n")
			}
		}

		// Success titles (cap at 5)
		if analytics.SuccessTitles != "" && analytics.SuccessTitles != "[]" {
			var titles []string
			if err := json.Unmarshal([]byte(analytics.SuccessTitles), &titles); err == nil && len(titles) > 0 {
				if len(titles) > 5 {
					titles = titles[:5]
				}
				sb.WriteString("\n### Títulos de sucesso (referência de estilo):\n")
				for _, t := range titles {
					sb.WriteString(fmt.Sprintf("- %s\n", t))
				}
			}
		}

		// Top tags (cap at 10)
		if analytics.TopTags != "" && analytics.TopTags != "[]" {
			var tags []TagCount
			if err := json.Unmarshal([]byte(analytics.TopTags), &tags); err == nil && len(tags) > 0 {
				if len(tags) > 10 {
					tags = tags[:10]
				}
				sb.WriteString("\n### Tags frequentes do canal:\n")
				tagParts := make([]string, 0, len(tags))
				for _, t := range tags {
					tagParts = append(tagParts, t.Tag)
				}
				sb.WriteString(strings.Join(tagParts, ", ") + "\n")
			}
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
				sb.WriteString(fmt.Sprintf("Destaque: %s\n", h.Text))
				sb.WriteString(fmt.Sprintf("Razão: %s\n", h.Reason))
				sb.WriteString(fmt.Sprintf("Score: %.1f\n", h.Score))
			}
		}
		// Add transcript segment for this clip
		if transcript != "" {
			segment := extractTranscriptSegment(transcript, clip.StartSec, clip.EndSec)
			if segment != "" {
				sb.WriteString(fmt.Sprintf("Transcrição: %s\n", segment))
			}
		}
		sb.WriteString("\n")
	}

	// Output format
	sb.WriteString("## Saída\n")
	sb.WriteString(fmt.Sprintf("Array JSON com exatamente %d objetos (mesma ordem dos clips acima):\n", len(clips)))
	sb.WriteString(`[{"thumbnail_text":"TEXTO IMPACTO","title":"...","description":"...","tags":["..."],"category_id":22}]`)
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
