package pipeline

import "fmt"

// AdvanceEvent describes which event triggered a state transition.
type AdvanceEvent string

const (
	// EventURLSubmitted — user submitted URL+channel at WAITING_URL
	EventURLSubmitted AdvanceEvent = "url_submitted"
	// EventModeSubmitted — user submitted ModeConfig at WAITING_MODE
	EventModeSubmitted AdvanceEvent = "mode_submitted"
	// EventExecutionDone — goroutine completed EXECUTING phase
	EventExecutionDone AdvanceEvent = "execution_done"
	// EventHighlightsReviewed — gate /review-highlights submitted
	EventHighlightsReviewed AdvanceEvent = "highlights_reviewed"
	// EventClipsGenerated — goroutine completed GENERATING_CLIPS phase
	EventClipsGenerated AdvanceEvent = "clips_generated"
	// EventThumbnailConfigSubmitted — gate /thumbnail-config submitted
	EventThumbnailConfigSubmitted AdvanceEvent = "thumbnail_config_submitted"
	// EventMetadataReviewed — gate /review-metadata submitted
	EventMetadataReviewed AdvanceEvent = "metadata_reviewed"
	// EventClipsReviewed — gate /review-clips submitted
	EventClipsReviewed AdvanceEvent = "clips_reviewed"
	// EventUploadConfirmed — gate /upload-confirm submitted
	EventUploadConfirmed AdvanceEvent = "upload_confirmed"
	// EventUploadDone — goroutine completed UPLOADING phase
	EventUploadDone AdvanceEvent = "upload_done"
	// EventError — any unrecoverable error
	EventError AdvanceEvent = "error"
	// EventCancel — user cancelled
	EventCancel AdvanceEvent = "cancel"
)

// Advance computes the next RunState from current + mode + event.
// It has no side effects — all persistence and SSE publishing is the
// executor's responsibility.
func Advance(current RunState, mode WorkflowMode, event AdvanceEvent) (RunState, error) {
	switch event {
	case EventCancel:
		return StateCancelled, nil
	case EventError:
		return StateError, nil
	}

	switch current {
	case StateWaitingURL:
		if event == EventURLSubmitted {
			return StateWaitingMode, nil
		}

	case StateWaitingMode:
		if event == EventModeSubmitted {
			return StateExecuting, nil
		}

	case StateExecuting:
		if event == EventExecutionDone {
			switch mode {
			case WorkflowAI:
				return StateWaitingReviewHighlights, nil
			case WorkflowLongform:
				return StateWaitingReviewMetadata, nil
			}
		}

	case StateWaitingReviewHighlights:
		if event == EventHighlightsReviewed {
			return StateGeneratingClips, nil
		}

	case StateGeneratingClips:
		if event == EventClipsGenerated {
			return StateWaitingThumbnailConfig, nil
		}

	case StateWaitingThumbnailConfig:
		if event == EventThumbnailConfigSubmitted {
			return StateWaitingReviewMetadata, nil
		}

	case StateWaitingReviewMetadata:
		if event == EventMetadataReviewed {
			return StateWaitingReviewClips, nil
		}

	case StateWaitingReviewClips:
		if event == EventClipsReviewed {
			return StateWaitingUploadConfirm, nil
		}

	case StateWaitingUploadConfirm:
		if event == EventUploadConfirmed {
			return StateUploading, nil
		}

	case StateUploading:
		if event == EventUploadDone {
			return StateDone, nil
		}
	}

	return current, fmt.Errorf("invalid transition: state=%s mode=%s event=%s", current, mode, event)
}
