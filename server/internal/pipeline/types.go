package pipeline

import "strconv"

// ── RunState ──────────────────────────────────────────────────────────────────

// RunState represents the current gate/execution state of a pipeline run.
type RunState string

const (
	StateWaitingURL              RunState = "WAITING_URL"
	StateWaitingMode             RunState = "WAITING_MODE"
	StateWaitingReviewHighlights RunState = "WAITING_REVIEW_HIGHLIGHTS"
	StateWaitingThumbnailConfig  RunState = "WAITING_THUMBNAIL_CONFIG"
	StateWaitingReviewMetadata   RunState = "WAITING_REVIEW_METADATA"
	StateWaitingReviewClips      RunState = "WAITING_REVIEW_CLIPS"
	StateWaitingUploadConfirm    RunState = "WAITING_UPLOAD_CONFIRM"
	StateExecuting               RunState = "EXECUTING"
	StateGeneratingClips         RunState = "GENERATING_CLIPS"
	StateUploading               RunState = "UPLOADING"
	StateDone                    RunState = "DONE"
	StateError                   RunState = "ERROR"
	StateCancelled               RunState = "CANCELLED"
)

// ── WorkflowMode ──────────────────────────────────────────────────────────────

// WorkflowMode defines which execution path to follow.
type WorkflowMode string

const (
	WorkflowAI       WorkflowMode = "ai"
	WorkflowLongform WorkflowMode = "longform"
)

// ── ExecutePhase ──────────────────────────────────────────────────────────────

// ExecutePhase is a sub-step within a compute state (for SSE sub-progress).
type ExecutePhase string

const (
	PhaseDownload   ExecutePhase = "download"
	PhaseTranscript ExecutePhase = "transcript"
	PhaseAnalyze    ExecutePhase = "analyze"
	PhaseSegment    ExecutePhase = "segment"
	PhaseCut        ExecutePhase = "cut"
	PhaseAntiDup    ExecutePhase = "anti_dup"
	PhaseLogo       ExecutePhase = "logo_watermark"
	PhaseShorts     ExecutePhase = "shorts"
	PhaseThumbnail  ExecutePhase = "thumbnail"
	PhaseUpload     ExecutePhase = "upload"
)

// ── SSE event type constants ──────────────────────────────────────────────────

const (
	SSEEventStateChanged  = "state_changed"
	SSEEventGateOpened    = "gate_opened"
	SSEEventPhaseProgress = "phase_progress"
	SSEEventDone          = "done"
	SSEEventError         = "error"
	SSEEventCancelled     = "cancelled"
)

// ── Run ───────────────────────────────────────────────────────────────────────

// Run is the in-memory/JSON representation of a pipeline_runs row.
type Run struct {
	ID              int64        `json:"id"`
	ChannelID       *int64       `json:"channel_id,omitempty"`
	URL             string       `json:"url"`
	Mode            WorkflowMode `json:"mode"`
	State           RunState     `json:"state"`
	ActivePhase     string       `json:"active_phase"`
	Error           string       `json:"error,omitempty"`
	ModeConfigJSON  string       `json:"-"`
	VideoPath       string       `json:"video_path,omitempty"`
	TranscriptPath  string       `json:"transcript_path,omitempty"`
	StartedAt       int64        `json:"started_at"`
	FinishedAt      *int64       `json:"finished_at,omitempty"`
}

// ── Highlight ─────────────────────────────────────────────────────────────────

// Highlight represents a detected highlight segment for AI mode review.
type Highlight struct {
	ID         int64   `json:"id"`
	RunID      int64   `json:"run_id"`
	StartSec   float64 `json:"start_sec"`
	EndSec     float64 `json:"end_sec"`
	AdjStartSec float64 `json:"adj_start_sec"`
	AdjEndSec   float64 `json:"adj_end_sec"`
	Score      float64 `json:"score"`
	Text       string  `json:"text"`
	Reason     string  `json:"reason"`
	IsSelected bool    `json:"is_selected"`
	CreatedAt  int64   `json:"created_at"`
}

// ── Clip ─────────────────────────────────────────────────────────────────────

// Clip represents a generated clip pending metadata review and upload.
type Clip struct {
	ID             int64   `json:"id"`
	RunID          int64   `json:"run_id"`
	HighlightID    *int64  `json:"highlight_id,omitempty"`
	FilePath       string  `json:"file_path"`
	ThumbnailPath  string  `json:"thumbnail_path"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Tags           string  `json:"tags"`
	ThumbnailStyle string  `json:"thumbnail_style"`
	IsSelected     bool    `json:"is_selected"`
	StartSec       float64 `json:"start_sec"`
	EndSec         float64 `json:"end_sec"`
	DurationSec    float64 `json:"duration_sec"`
	UploadStatus   string  `json:"upload_status"`
	YoutubeID      string  `json:"youtube_id,omitempty"`
	CreatedAt      int64   `json:"created_at"`
}

// ── DownloadInfo ──────────────────────────────────────────────────────────────

// DownloadInfo carries sub-progress within the download phase.
type DownloadInfo struct {
	PercentDone float64 `json:"percent_done"`
	SpeedKBs    float64 `json:"speed_kbs"`
	ETASec      int     `json:"eta_sec"`
}

// ── ModeConfig ────────────────────────────────────────────────────────────────

// ModeConfig captures the full configuration submitted at WAITING_MODE.
type ModeConfig struct {
	Mode            WorkflowMode `json:"mode"`
	MinDurationSecs int          `json:"min_duration_secs"`
	MaxDurationSecs int          `json:"max_duration_secs"`
	SensitivityPct  int          `json:"sensitivity_pct,omitempty"` // AI only
	SegmentSecs     int          `json:"segment_secs,omitempty"`    // longform only
}

// ── Gate request types ────────────────────────────────────────────────────────

// AdvanceRequest is the body for POST /runs/{id}/advance.
type AdvanceRequest struct {
	URL       string      `json:"url,omitempty"`
	ChannelID *int64      `json:"channel_id,omitempty"`
	Mode      *ModeConfig `json:"mode_config,omitempty"`
}

// ReviewHighlightsRequest is the body for POST /runs/{id}/gates/review-highlights.
type ReviewHighlightsRequest struct {
	Highlights []HighlightUpdate `json:"highlights"`
}

// HighlightUpdate carries user edits for a single highlight.
type HighlightUpdate struct {
	ID          int64   `json:"id"`
	AdjStartSec float64 `json:"adj_start_sec"`
	AdjEndSec   float64 `json:"adj_end_sec"`
	IsSelected  bool    `json:"is_selected"`
}

// ThumbnailConfigRequest is the body for POST /runs/{id}/gates/thumbnail-config.
type ThumbnailConfigRequest struct {
	DefaultStyle string          `json:"default_style"`
	ClipOverrides []ClipThumbnailOverride `json:"clip_overrides,omitempty"`
}

// ClipThumbnailOverride allows per-clip thumbnail style selection.
type ClipThumbnailOverride struct {
	ClipID int64  `json:"clip_id"`
	Style  string `json:"style"`
}

// ReviewMetadataRequest is the body for POST /runs/{id}/gates/review-metadata.
type ReviewMetadataRequest struct {
	Clips []ClipMetadataUpdate `json:"clips"`
}

// ClipMetadataUpdate carries user edits for a clip's metadata.
type ClipMetadataUpdate struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
}

// ReviewClipsRequest is the body for POST /runs/{id}/gates/review-clips.
type ReviewClipsRequest struct {
	SelectedIDs []int64 `json:"selected_ids"`
}

// UploadConfirmRequest is the body for POST /runs/{id}/gates/upload-confirm.
type UploadConfirmRequest struct {
	Privacy string `json:"privacy"` // "public" | "unlisted" | "private"
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// PipelineJobID returns the deterministic SSE job ID for a pipeline run.
func PipelineJobID(runID int64) string {
	return "pipeline-" + strconv.FormatInt(runID, 10)
}
