package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/transcript"
)

// TranscriptHandler handles video transcription requests.
type TranscriptHandler struct {
	hub         *hub.SSEHub
	transcriber *transcript.WhisperTranscriber
	cache       *transcript.TranscriptCache
	log         *slog.Logger
}

// NewTranscriptHandler creates a TranscriptHandler.
func NewTranscriptHandler(
	h *hub.SSEHub,
	transcriber *transcript.WhisperTranscriber,
	cache *transcript.TranscriptCache,
) *TranscriptHandler {
	return &TranscriptHandler{
		hub:         h,
		transcriber: transcriber,
		cache:       cache,
		log:         slog.With("component", "api", "handler", "transcript"),
	}
}

type transcriptRequest struct {
	VideoPath string `json:"video_path"`
	Language  string `json:"language"`
}

// PostTranscript handles POST /api/transcript.
func (h *TranscriptHandler) PostTranscript(w http.ResponseWriter, r *http.Request) {
	var req transcriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.VideoPath == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "video_path is required")
		return
	}

	jobID := newJobID()
	h.log.Info("transcript job started", "jobID", jobID, "path", req.VideoPath)
	go h.runTranscript(jobID, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *TranscriptHandler) runTranscript(jobID string, req transcriptRequest) {
	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]interface{}{"chunk": 1, "total": 1, "status": "starting"}})

	// Check cache first
	hash, err := h.cache.Hash(req.VideoPath)
	if err == nil {
		if cached, ok := h.cache.Get(hash); ok {
			h.log.Info("transcript cache hit", "jobID", jobID, "hash", hash)
			h.hub.Publish(jobID, hub.SSEEvent{
				Type: "done",
				Data: map[string]interface{}{"segments": len(cached.Segments), "cached": true},
			})
			return
		}
	}

	h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]interface{}{"status": "transcribing"}})

	t, err := h.transcriber.Transcribe(req.VideoPath)
	if err != nil {
		h.log.Error("transcription failed", "jobID", jobID, "err", err)
		h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
		return
	}

	// Store in cache (best-effort)
	if hash != "" {
		if putErr := h.cache.Put(hash, t); putErr != nil {
			h.log.Warn("transcript cache put failed", "jobID", jobID, "err", putErr)
		}
	}

	h.hub.Publish(jobID, hub.SSEEvent{
		Type: "done",
		Data: map[string]interface{}{
			"segments": len(t.Segments),
			"language": t.Language,
		},
	})
}

// GetTranscriptStream handles GET /api/transcript/{id}/stream.
func (h *TranscriptHandler) GetTranscriptStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}
