package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// MetadataHandler handles HTTP requests for AI-powered clip metadata generation.
type MetadataHandler struct {
	runRepo   *database.PipelineRunRepo
	clipRepo  *database.PipelineClipRepo
	generator *ai.MetadataGenerator
	hub       *hub.SSEHub
	log       *slog.Logger
}

// NewMetadataHandler constructs a MetadataHandler.
func NewMetadataHandler(
	runRepo *database.PipelineRunRepo,
	clipRepo *database.PipelineClipRepo,
	generator *ai.MetadataGenerator,
	h *hub.SSEHub,
) *MetadataHandler {
	return &MetadataHandler{
		runRepo:   runRepo,
		clipRepo:  clipRepo,
		generator: generator,
		hub:       h,
		log:       slog.With("handler", "metadata"),
	}
}

// PostBatchGenerate starts batch metadata generation for all clips in a run.
// POST /api/metadata/runs/{id}/generate
func (g *MetadataHandler) PostBatchGenerate(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}

	// Validate run state
	run, err := g.runRepo.GetByID(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, "run not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to load run", http.StatusInternalServerError)
		return
	}
	if run.State != "WAITING_REVIEW_METADATA" {
		jsonError(w, fmt.Sprintf("run not in WAITING_REVIEW_METADATA state (current: %s)", run.State), http.StatusConflict)
		return
	}

	// Check Claude CLI availability
	if !g.generator.Available() {
		jsonError(w, "Claude CLI not available — install with 'npm install -g @anthropic-ai/claude-code'", http.StatusServiceUnavailable)
		return
	}

	// Load clips to get count
	clips, err := g.clipRepo.ListByRun(r.Context(), runID)
	if err != nil {
		jsonError(w, "failed to load clips", http.StatusInternalServerError)
		return
	}
	if len(clips) == 0 {
		jsonError(w, "no clips found for this run", http.StatusBadRequest)
		return
	}

	// Launch async generation
	go g.generator.GenerateBatch(context.Background(), runID)

	// Return 202 immediately
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":      fmt.Sprintf("meta_%d", runID),
		"total_clips": len(clips),
	})
}

// PostSingleGenerate generates metadata for a single clip.
// POST /api/metadata/runs/{id}/clips/{clipId}/generate
func (g *MetadataHandler) PostSingleGenerate(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	clipIDStr := r.PathValue("clipId")
	clipID, err := strconv.ParseInt(clipIDStr, 10, 64)
	if err != nil || clipID <= 0 {
		jsonError(w, "invalid clip id", http.StatusBadRequest)
		return
	}

	// Validate run state
	run, err := g.runRepo.GetByID(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, "run not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to load run", http.StatusInternalServerError)
		return
	}
	if run.State != "WAITING_REVIEW_METADATA" {
		jsonError(w, fmt.Sprintf("run not in WAITING_REVIEW_METADATA state (current: %s)", run.State), http.StatusConflict)
		return
	}

	// Check Claude CLI availability
	if !g.generator.Available() {
		jsonError(w, "Claude CLI not available — install with 'npm install -g @anthropic-ai/claude-code'", http.StatusServiceUnavailable)
		return
	}

	// Generate synchronously
	metadata, err := g.generator.GenerateSingle(r.Context(), runID, clipID)
	if err != nil {
		g.log.Error("single metadata generation failed", "run_id", runID, "clip_id", clipID, "err", err)
		jsonError(w, fmt.Sprintf("generation failed: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}

// clipMetadataEdit represents a single clip's metadata edit from the review UI.
type clipMetadataEdit struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Tags          string `json:"tags"`
	ThumbnailText string `json:"thumbnail_text"`
}

// reviewMetadataRequest is the request body for PostReviewMetadata.
type reviewMetadataRequest struct {
	Clips []clipMetadataEdit `json:"clips"`
}

// PostReviewMetadata saves edited metadata and advances the pipeline.
// POST /api/pipeline/runs/{id}/gates/review-metadata
func (g *MetadataHandler) PostReviewMetadata(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}

	// Validate run state
	run, err := g.runRepo.GetByID(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, "run not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to load run", http.StatusInternalServerError)
		return
	}
	if run.State != "WAITING_REVIEW_METADATA" {
		jsonError(w, fmt.Sprintf("run not in WAITING_REVIEW_METADATA state (current: %s)", run.State), http.StatusConflict)
		return
	}

	// Parse request body
	var req reviewMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Reject placeholder titles (empty or still "Part N")
	placeholderRe := regexp.MustCompile(`(?i)^(part\s*\d+)?$`)
	for _, clip := range req.Clips {
		if placeholderRe.MatchString(strings.TrimSpace(clip.Title)) {
			jsonError(w, fmt.Sprintf("clip %d has no title — generate metadata or enter a title before continuing", clip.ID), http.StatusBadRequest)
			return
		}
	}

	// Update each clip's metadata
	for _, clip := range req.Clips {
		if err := g.clipRepo.UpdateMetadata(r.Context(), clip.ID, clip.Title, clip.Description, clip.Tags, clip.ThumbnailText); err != nil {
			g.log.Warn("failed to update clip metadata", "clip_id", clip.ID, "err", err)
			jsonError(w, fmt.Sprintf("failed to update clip %d: %s", clip.ID, err), http.StatusInternalServerError)
			return
		}
	}

	// Advance pipeline state
	if err := g.runRepo.AdvanceState(r.Context(), runID, "WAITING_REVIEW_METADATA", "WAITING_REVIEW_CLIPS"); err != nil {
		jsonError(w, fmt.Sprintf("failed to advance state: %s", err), http.StatusInternalServerError)
		return
	}

	// Publish SSE state_changed event
	jobKey := fmt.Sprintf("%d", runID)
	g.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: map[string]interface{}{
			"run_id": runID,
			"state":  "WAITING_REVIEW_CLIPS",
		},
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"state": "WAITING_REVIEW_CLIPS",
	})
}
