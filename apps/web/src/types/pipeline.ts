// ── RunState ──────────────────────────────────────────────────────────────────

export type RunState =
  | 'WAITING_URL'
  | 'WAITING_MODE'
  | 'WAITING_REVIEW_HIGHLIGHTS'
  | 'WAITING_THUMBNAIL_CONFIG'
  | 'WAITING_REVIEW_METADATA'
  | 'WAITING_REVIEW_CLIPS'
  | 'WAITING_UPLOAD_CONFIRM'
  | 'EXECUTING'
  | 'GENERATING_CLIPS'
  | 'UPLOADING'
  | 'DONE'
  | 'ERROR'
  | 'CANCELLED';

// ── WorkflowMode ──────────────────────────────────────────────────────────────

export type WorkflowMode = 'ai' | 'longform';

// ── ExecutePhase ──────────────────────────────────────────────────────────────

export type ExecutePhase =
  | 'download'
  | 'transcript'
  | 'analyze'
  | 'segment'
  | 'cut'
  | 'shorts'
  | 'thumbnail'
  | 'upload';

// ── Run ───────────────────────────────────────────────────────────────────────

export interface Run {
  id: number;
  channel_id: number | null;
  url: string;
  mode: WorkflowMode;
  state: RunState;
  active_phase: string;
  error: string;
  video_path: string;
  transcript_path: string;
  started_at: number;
  finished_at: number | null;
}

// ── Highlight ─────────────────────────────────────────────────────────────────

export interface Highlight {
  id: number;
  run_id: number;
  start_sec: number;
  end_sec: number;
  adj_start_sec: number;
  adj_end_sec: number;
  score: number;
  text: string;
  reason: string;
  is_selected: boolean;
  created_at: number;
}

// ── Clip ─────────────────────────────────────────────────────────────────────

export interface Clip {
  id: number;
  run_id: number;
  highlight_id: number | null;
  file_path: string;
  thumbnail_path: string;
  title: string;
  description: string;
  tags: string;
  thumbnail_style: string;
  is_selected: boolean;
  start_sec: number;
  end_sec: number;
  duration_sec: number;
  upload_status: string;
  youtube_id: string;
  created_at: number;
}

// ── ModeConfig ────────────────────────────────────────────────────────────────

export interface ModeConfig {
  mode: WorkflowMode;
  min_duration_secs: number;
  max_duration_secs: number;
  sensitivity_pct?: number;   // AI only
  segment_secs?: number;      // longform only
}

// ── Gate request types ────────────────────────────────────────────────────────

export interface AdvanceRequest {
  url?: string;
  channel_id?: number;
  mode_config?: ModeConfig;
}

export interface HighlightUpdate {
  id: number;
  adj_start_sec: number;
  adj_end_sec: number;
  is_selected: boolean;
}

export interface ReviewHighlightsRequest {
  highlights: HighlightUpdate[];
}

export interface ClipThumbnailOverride {
  clip_id: number;
  style: string;
}

export interface ThumbnailConfigRequest {
  default_style: string;
  clip_overrides?: ClipThumbnailOverride[];
}

export interface ClipMetadataUpdate {
  id: number;
  title: string;
  description: string;
  tags: string;
}

export interface ReviewMetadataRequest {
  clips: ClipMetadataUpdate[];
}

export interface ReviewClipsRequest {
  selected_ids: number[];
}

export interface UploadConfirmRequest {
  privacy: 'public' | 'unlisted' | 'private';
}

// ── SSE event payloads ────────────────────────────────────────────────────────

export interface SSEStateChangedPayload {
  run_id: number;
  state: RunState;
}

export interface SSEGateOpenedPayload {
  run_id: number;
  state: RunState;
}

export interface SSEPhaseProgressPayload {
  run_id: number;
  phase: ExecutePhase;
  percent_done: number;
  speed_kbs?: number;
  eta_sec?: number;
  clip_id?: number;
  youtube_id?: string;
  warning?: string;
}

export interface SSEEvent {
  type:
    | 'state_changed'
    | 'gate_opened'
    | 'phase_progress'
    | 'done'
    | 'error'
    | 'cancelled'
    | 'ping';
  data?:
    | SSEStateChangedPayload
    | SSEGateOpenedPayload
    | SSEPhaseProgressPayload
    | { run_id: number; state: RunState; message?: string };
}

// ── Step rail helpers ─────────────────────────────────────────────────────────

export type StepStatus = 'idle' | 'running' | 'gate' | 'done' | 'error';

export interface StepRailEntry {
  label: string;
  state: RunState;
  status: StepStatus;
}
