package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/transcript"
)

// AIHandler handles AI topic analysis requests.
type AIHandler struct {
	hub      *hub.SSEHub
	detector *ai.TopicTransitionDetector
	log      *slog.Logger
}

// NewAIHandler creates an AIHandler.
func NewAIHandler(h *hub.SSEHub, detector *ai.TopicTransitionDetector) *AIHandler {
	return &AIHandler{
		hub:      h,
		detector: detector,
		log:      slog.With("component", "api", "handler", "ai"),
	}
}

type analyzeRequest struct {
	TranscriptJSON string `json:"transcript_json"`
}

// PostAnalyze handles POST /api/analyze.
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

	jobID := newJobID()
	h.log.Info("analyze job started", "jobID", jobID, "segments", len(t.Segments))
	go h.runAnalyze(r.Context(), jobID, t)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

// transcriptSegmentAdapter adapts transcript.Segment to ai.TranscriptSegment.
type transcriptSegmentAdapter struct {
	seg transcript.Segment
}

func (a transcriptSegmentAdapter) GetStart() float64 { return a.seg.Start.Seconds() }
func (a transcriptSegmentAdapter) GetEnd() float64   { return a.seg.End.Seconds() }
func (a transcriptSegmentAdapter) GetText() string   { return a.seg.Text }

func (h *AIHandler) runAnalyze(ctx context.Context, jobID string, t transcript.Transcript) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "analyzing"}})

	// Convert transcript.Segment slice to []ai.TranscriptSegment
	segs := make([]ai.TranscriptSegment, len(t.Segments))
	for i, s := range t.Segments {
		segs[i] = transcriptSegmentAdapter{seg: s}
	}

	result, err := h.detector.Analyze(ctx, segs)
	if err != nil {
		h.log.Error("ai analysis failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
		return
	}

	// Publish each highlight as a streaming event
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

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{
			"topics":     len(result.Topics),
			"highlights": len(result.Highlights),
		},
	})
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
