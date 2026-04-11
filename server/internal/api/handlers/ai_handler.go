package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/transcript"
)

// AIHandler handles AI topic analysis requests.
type AIHandler struct {
	hub          *hub.SSEHub
	detector     *ai.TopicTransitionDetector
	pipelineRepo *database.PipelineRunRepo
	ffmpegPath   string
	log          *slog.Logger
}

// NewAIHandler creates an AIHandler.
// ffmpegPath is the resolved path to the ffmpeg binary (from configurator).
// Pass an empty string to fall back to "ffmpeg" on PATH.
func NewAIHandler(h *hub.SSEHub, detector *ai.TopicTransitionDetector, pipelineRepo *database.PipelineRunRepo, ffmpegPath string) *AIHandler {
	return &AIHandler{
		hub:          h,
		detector:     detector,
		pipelineRepo: pipelineRepo,
		ffmpegPath:   ffmpegPath,
		log:          slog.With("component", "api", "handler", "ai"),
	}
}

// analyzeRequest is the POST /api/analyze request body.
//
// Backward-compatible extension:
//   - Strategies: list of detection strategies to use. Default ["ai"] when omitted.
//   - ChatJSONPath: path to TwitchDownloaderCLI JSON chat export (required for "chat" strategy).
//   - VideoPath: path to the video file (required for "audio" and "visual" strategies).
type analyzeRequest struct {
	TranscriptJSON string   `json:"transcript_json"`
	SessionID      string   `json:"session_id"`
	Strategies     []string `json:"strategies"`      // optional; default: ["ai"]
	ChatJSONPath   string   `json:"chat_json_path"`  // optional; used by "chat" strategy
	VideoPath      string   `json:"video_path"`      // optional; used by "audio" and "visual" strategies
}

// PostAnalyze handles POST /api/analyze.
//
// When strategies is omitted or ["ai"], the response is identical to the
// original behaviour ([]Highlight via SSE). When multiple strategies are
// requested, the response uses []UnifiedHighlight with confidence scoring.
func (h *AIHandler) PostAnalyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.TranscriptJSON == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "transcript_json is required")
		return
	}

	// Parse transcript segments from JSON
	var t transcript.Transcript
	if err := json.Unmarshal([]byte(req.TranscriptJSON), &t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_transcript", err.Error())
		return
	}

	// Normalise strategies: default to ["ai"] for backward compatibility.
	strategies := normaliseStrategies(req.Strategies)

	jobID := newJobID()
	h.log.Info("analyze job started", "jobID", jobID, "segments", len(t.Segments), "strategies", strategies)
	go h.runAnalyze(r.Context(), jobID, req, t, strategies)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

// transcriptSegmentAdapter adapts transcript.Segment to ai.TranscriptSegment.
type transcriptSegmentAdapter struct {
	seg transcript.Segment
}

func (a transcriptSegmentAdapter) GetStart() float64 { return a.seg.Start.Seconds() }
func (a transcriptSegmentAdapter) GetEnd() float64   { return a.seg.End.Seconds() }
func (a transcriptSegmentAdapter) GetText() string   { return a.seg.Text }

// runAnalyze is the goroutine that drives the full multi-strategy analysis.
func (h *AIHandler) runAnalyze(
	ctx context.Context,
	jobID string,
	req analyzeRequest,
	t transcript.Transcript,
	strategies []ai.Strategy,
) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "analyzing"}})

	// Convert transcript segments to ai.TranscriptSegment.
	segs := make([]ai.TranscriptSegment, len(t.Segments))
	for i, s := range t.Segments {
		segs[i] = transcriptSegmentAdapter{seg: s}
	}

	// Determine whether we are in single-AI mode (backward compat) or multi-strategy mode.
	aiOnly := len(strategies) == 1 && strategies[0] == ai.StrategyAI

	if aiOnly {
		h.runAIOnly(ctx, jobID, req, segs)
		return
	}

	h.runMultiStrategy(ctx, jobID, req, segs, strategies)
}

// runAIOnly is the original single-strategy path — kept identical to the
// previous implementation for full backward compatibility.
func (h *AIHandler) runAIOnly(
	ctx context.Context,
	jobID string,
	req analyzeRequest,
	segs []ai.TranscriptSegment,
) {
	result, err := h.detector.Analyze(ctx, segs)
	if err != nil {
		h.log.Error("ai analysis failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
		return
	}

	// Publish each highlight as a streaming event.
	for _, hl := range result.Highlights {
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "highlight",
			Data: map[string]interface{}{
				"start": hl.StartSec,
				"end":   hl.EndSec,
				"score": hl.Score,
			},
		})
	}

	data := map[string]interface{}{
		"topics":     len(result.Topics),
		"highlights": len(result.Highlights),
		"success":    true,
	}
	h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
	if req.SessionID != "" {
		persistStepOutput(h.pipelineRepo, req.SessionID, "analyze", data)
	}
}

// runMultiStrategy runs all requested strategies in parallel and aggregates results.
func (h *AIHandler) runMultiStrategy(
	ctx context.Context,
	jobID string,
	req analyzeRequest,
	segs []ai.TranscriptSegment,
	strategies []ai.Strategy,
) {
	var (
		mu              sync.Mutex
		wg              sync.WaitGroup
		aiHighlights    []ai.Highlight
		audioHighlights []ai.AudioHighlight
		visualChanges   []ai.SceneChange
		chatHighlights  []ai.ChatHighlight
	)

	activeSet := make(map[ai.Strategy]bool, len(strategies))
	for _, s := range strategies {
		activeSet[s] = true
	}

	// AI strategy.
	if activeSet[ai.StrategyAI] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := h.detector.Analyze(ctx, segs)
			if err != nil {
				h.log.Error("ai strategy failed", "jobID", jobID, "err", err)
				return
			}
			mu.Lock()
			aiHighlights = result.Highlights
			mu.Unlock()
		}()
	}

	// Audio strategy.
	if activeSet[ai.StrategyAudio] {
		if req.VideoPath == "" {
			h.log.Warn("audio strategy requested but video_path is empty — skipping", "jobID", jobID)
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				det := ai.NewAudioDetector(h.ffmpegPath)
				hls, err := det.DetectHighlights(ctx, req.VideoPath)
				if err != nil {
					h.log.Error("audio strategy failed", "jobID", jobID, "err", err)
					return
				}
				mu.Lock()
				audioHighlights = hls
				mu.Unlock()
			}()
		}
	}

	// Visual strategy.
	if activeSet[ai.StrategyVisual] {
		if req.VideoPath == "" {
			h.log.Warn("visual strategy requested but video_path is empty — skipping", "jobID", jobID)
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				det := ai.NewVisualDetector(h.ffmpegPath)
				scs, err := det.DetectSceneChanges(ctx, req.VideoPath)
				if err != nil {
					h.log.Error("visual strategy failed", "jobID", jobID, "err", err)
					return
				}
				mu.Lock()
				visualChanges = scs
				mu.Unlock()
			}()
		}
	}

	// Chat strategy.
	if activeSet[ai.StrategyChat] {
		if req.ChatJSONPath == "" {
			h.log.Warn("chat strategy requested but chat_json_path is empty — skipping", "jobID", jobID)
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				det := ai.NewChatActivityDetector()
				hls, err := det.DetectHighlights(ctx, req.ChatJSONPath)
				if err != nil {
					h.log.Error("chat strategy failed", "jobID", jobID, "err", err)
					return
				}
				mu.Lock()
				chatHighlights = hls
				mu.Unlock()
			}()
		}
	}

	wg.Wait()

	// Aggregate.
	aggregator := ai.NewHighlightAggregator()
	unified := aggregator.Aggregate(aiHighlights, audioHighlights, visualChanges, chatHighlights, strategies)

	// Publish each unified highlight as a streaming event.
	for _, uh := range unified {
		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "highlight",
			Data: map[string]interface{}{
				"start":      uh.StartSec,
				"end":        uh.EndSec,
				"confidence": uh.Confidence,
				"strategies": uh.Strategies,
				"scores":     uh.Scores,
				"type":       uh.Type,
			},
		})
	}

	data := map[string]interface{}{
		"unified_highlights": len(unified),
		"strategies_used":    strategies,
		"success":            true,
	}
	h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: data})
	if req.SessionID != "" {
		persistStepOutput(h.pipelineRepo, req.SessionID, "analyze", data)
	}
}

// GetAnalyzeStream handles GET /api/analyze/{id}/stream.
func (h *AIHandler) GetAnalyzeStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}

// normaliseStrategies converts a raw []string into []ai.Strategy.
// Returns [StrategyAI] when the input is nil or empty (backward compatibility).
func normaliseStrategies(raw []string) []ai.Strategy {
	if len(raw) == 0 {
		return []ai.Strategy{ai.StrategyAI}
	}
	out := make([]ai.Strategy, 0, len(raw))
	for _, s := range raw {
		switch ai.Strategy(s) {
		case ai.StrategyAI, ai.StrategyAudio, ai.StrategyVisual, ai.StrategyChat:
			out = append(out, ai.Strategy(s))
		default:
			slog.Warn("analyzeRequest: unknown strategy ignored", "strategy", s)
		}
	}
	if len(out) == 0 {
		return []ai.Strategy{ai.StrategyAI}
	}
	return out
}
