package pipeline

// RunState enumerates valid pipeline run states.
// Must stay in sync with the frontend RunState type in apps/web/src/types/pipeline.ts.
const (
	StateWaitingURL            = "WAITING_URL"
	StateExecuting             = "EXECUTING"
	StateWaitingMode           = "WAITING_MODE"
	StateWaitingReviewHighlights = "WAITING_REVIEW_HIGHLIGHTS"
	StateWaitingThumbnailConfig  = "WAITING_THUMBNAIL_CONFIG"
	StateWaitingReviewMetadata   = "WAITING_REVIEW_METADATA"
	StateWaitingReviewClips      = "WAITING_REVIEW_CLIPS"
	StateWaitingUploadConfirm    = "WAITING_UPLOAD_CONFIRM"
	StateGeneratingClips         = "GENERATING_CLIPS"
	StateUploading               = "UPLOADING"
	StateDone                    = "DONE"
	StateError                   = "ERROR"
	StateCancelled               = "CANCELLED"
)

// validTransitions maps each non-terminal state to its successor gate state.
// Used by Advance() to determine the next state for each dispatch branch.
var validTransitions = map[string]string{
	StateWaitingURL:              StateWaitingMode,
	StateWaitingMode:             StateExecuting,
	StateExecuting:               StateWaitingReviewHighlights,
	StateWaitingReviewHighlights: StateGeneratingClips,
	StateGeneratingClips:         StateWaitingThumbnailConfig,
	StateWaitingThumbnailConfig:  StateWaitingReviewMetadata,
	StateWaitingReviewMetadata:   StateWaitingReviewClips,
	StateWaitingReviewClips:      StateWaitingUploadConfirm,
	StateWaitingUploadConfirm:    StateUploading,
	StateUploading:               StateDone,
}

// PhaseID enumerates the active_phase values used during EXECUTING states.
const (
	PhaseDownload   = "download"
	PhaseTranscribe = "transcribe"
	PhaseAnalyze    = "analyze"
	PhaseSegment    = "segment"
	PhaseCut        = "cut"
	PhaseShorts     = "shorts"
	PhaseThumbnail  = "thumbnail"
	PhaseUpload     = "upload"
)
