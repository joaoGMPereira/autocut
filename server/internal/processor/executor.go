package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// Pipeline state constants — duplicated here to avoid circular import with pipeline package.
// Must stay in sync with pipeline/types.go.
const (
	stateExecuting               = "EXECUTING"
	stateGeneratingClips         = "GENERATING_CLIPS"
	stateWaitingReviewHighlights = "WAITING_REVIEW_HIGHLIGHTS"
)

// ExecRequest is the input to RunExecution.
// All dependencies are passed as function values to avoid circular imports.
type ExecRequest struct {
	RunID       int64
	VideoPath   string
	VideoURL    string
	DurationSec int64
	DataDir     string
	Mode        string // "ai" (default) or "longform"

	// Progress/event publishing (bound to hub.Publish in service.go)
	PublishPhaseProgress func(phase string, pct float64)
	PublishStateChange   func(state string)
	PublishError         func(msg string)

	// Repo operations (bound to concrete repo methods in service.go)
	FindTranscriptByURL func(ctx context.Context, url string) (string, error)
	SetTranscriptPath   func(ctx context.Context, id int64, path string) error
	AdvanceState        func(ctx context.Context, id int64, from, to string) error
	Finish              func(ctx context.Context, id int64, state, errMsg string) error
	ReplaceHighlights   func(ctx context.Context, runID int64, highlights []database.PipelineHighlight) error
}

// RunExecution orchestrates the full transcription + highlight detection pipeline
// for a run that is already in EXECUTING state.
// On success, transitions to WAITING_REVIEW_HIGHLIGHTS.
// On any non-recoverable error, transitions to ERROR and publishes the error via SSE.
func RunExecution(ctx context.Context, req ExecRequest) error {
	log := slog.With("run_id", req.RunID, "phase", "execution")

	publishError := func(msg string) {
		log.Error("execution failed", "err", msg)
		_ = req.Finish(ctx, req.RunID, "ERROR", msg)
		req.PublishError(msg)
	}

	// Step 1: Verify video file exists on disk.
	if _, statErr := os.Stat(req.VideoPath); statErr != nil {
		msg := fmt.Sprintf("video file not found: %s", req.VideoPath)
		publishError(msg)
		return fmt.Errorf("%s", msg)
	}

	// Step 2: Find Whisper model.
	modelPath := FindWhisperModel(req.DataDir)
	if modelPath == "" {
		msg := "whisper model not found — install a ggml-*.bin model in {dataDir}/models/whisper/"
		publishError(msg)
		return fmt.Errorf("%s", msg)
	}
	log.Info("whisper model found", "model_path", modelPath)

	// Step 3: Transcript cache check or fresh transcription.
	transcriptPath, cacheErr := req.FindTranscriptByURL(ctx, req.VideoURL)
	if cacheErr != nil {
		log.Warn("transcript cache lookup failed, proceeding with fresh transcription", "err", cacheErr)
		transcriptPath = ""
	}

	if transcriptPath != "" {
		// Verify the cached file still exists on disk.
		if _, statErr := os.Stat(transcriptPath); statErr != nil {
			log.Warn("cached transcript file missing, re-transcribing", "path", transcriptPath)
			transcriptPath = ""
		}
	}

	if transcriptPath != "" {
		log.Info("transcript cache hit", "path", transcriptPath)
		// Emit 100% immediately for the transcript phase.
		req.PublishPhaseProgress("transcript", 100)
	} else {
		log.Info("transcribing full video", "video_path", req.VideoPath)

		onTranscriptProgress := func(pct float64) {
			req.PublishPhaseProgress("transcript", pct)
		}

		var cleanup func()
		var transcribeErr error
		transcriptPath, cleanup, transcribeErr = TranscribeFullVideo(
			req.VideoPath,
			modelPath,
			req.DataDir,
			req.DurationSec,
			onTranscriptProgress,
			ctx,
		)
		if cleanup != nil {
			defer cleanup()
		}
		if transcribeErr != nil {
			msg := fmt.Sprintf("transcription failed: %s", transcribeErr)
			publishError(msg)
			return fmt.Errorf("%s", msg)
		}
	}

	// Persist transcript path (both fresh transcription and cache hit).
	if setErr := req.SetTranscriptPath(ctx, req.RunID, transcriptPath); setErr != nil {
		log.Warn("failed to persist transcript_path", "err", setErr)
		// Non-fatal — continue with highlight detection.
	}

	// Step 4: Parse whisper JSON output.
	data, readErr := os.ReadFile(transcriptPath)
	if readErr != nil {
		msg := fmt.Sprintf("read transcript JSON: %s", readErr)
		publishError(msg)
		return fmt.Errorf("%s", msg)
	}

	var whisperResult whisperOutput
	if parseErr := json.Unmarshal(data, &whisperResult); parseErr != nil {
		msg := fmt.Sprintf("parse transcript JSON: %s", parseErr)
		publishError(msg)
		return fmt.Errorf("%s", msg)
	}

	// Convert to HighlightSegment slice (seconds from milliseconds).
	hlSegments := make([]HighlightSegment, 0, len(whisperResult.Transcription))
	for _, s := range whisperResult.Transcription {
		seg := HighlightSegment{
			StartSec: float64(s.Offsets.From) / 1000.0,
			EndSec:   float64(s.Offsets.To) / 1000.0,
			Text:     s.Text,
		}
		if seg.EndSec > seg.StartSec {
			hlSegments = append(hlSegments, seg)
		}
	}
	log.Info("transcript parsed", "segments", len(hlSegments))

	// Longform mode: skip highlight detection and advance directly to GENERATING_CLIPS.
	if req.Mode == "longform" {
		if advErr := req.AdvanceState(ctx, req.RunID, stateExecuting, stateGeneratingClips); advErr != nil {
			log.Warn("advance state failed (may already have advanced)", "err", advErr)
		}
		req.PublishStateChange(stateGeneratingClips)
		log.Info("execution complete (longform) — advancing to clip generation")
		return nil
	}

	// Step 5: Detect highlights (AI mode only).
	onAnalyzeProgress := func(pct float64) {
		req.PublishPhaseProgress("analyze", pct)
	}
	highlights := DetectHighlights(hlSegments, onAnalyzeProgress)

	// Stamp each highlight with the run ID.
	for i := range highlights {
		highlights[i].RunID = req.RunID
	}
	log.Info("highlights detected", "count", len(highlights))

	// Step 5b: Replace highlights in DB (atomic delete + insert, even if empty slice).
	if replaceErr := req.ReplaceHighlights(ctx, req.RunID, highlights); replaceErr != nil {
		msg := fmt.Sprintf("replace highlights: %s", replaceErr)
		publishError(msg)
		return fmt.Errorf("%s", msg)
	}

	// Step 6: Advance state to WAITING_REVIEW_HIGHLIGHTS.
	if advErr := req.AdvanceState(ctx, req.RunID, stateExecuting, stateWaitingReviewHighlights); advErr != nil {
		log.Warn("advance state failed (may already have advanced)", "err", advErr)
		// Non-fatal if run has already advanced (idempotent).
	}
	req.PublishStateChange(stateWaitingReviewHighlights)

	log.Info("execution complete — awaiting highlight review", "highlights", len(highlights))
	return nil
}
