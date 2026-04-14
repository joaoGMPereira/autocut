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

// ── AntiDuplicateConfig ──────────────────────────────────────────────────────
export interface AntiDupEffects {
  speed_boost?: boolean;
  crop?: boolean;
  color_grading?: boolean;
  noise?: boolean;
  noise_strength?: number;    // 1–10
  blur?: boolean;
  blur_edge_pct?: number;     // 0–100
  zoom?: boolean;
  transitions?: boolean;
}

export interface AntiDuplicateConfig {
  enabled: boolean;
  mode: 'subtle' | 'aggressive';
  effects?: AntiDupEffects;
}

// ── ModeConfig ────────────────────────────────────────────────────────────────

export interface ModeConfig {
  mode: WorkflowMode;
  min_duration_secs: number;
  max_duration_secs: number;
  sensitivity_pct?: number;        // AI only — highlights threshold 0–100
  segment_secs?: number;           // longform only — target part duration
  skip_music_mins?: number | null; // AI only — skip leading music in minutes
  force_regenerate?: boolean;      // AI only — ignore cached transcript/analysis
  skip_transcription?: boolean;    // AI only — skip Whisper, use existing transcript
  anti_duplicate?: AntiDuplicateConfig; // All modes — anti-dup protection config
  skip_regenerate?: boolean;            // All modes — skip reprocessing if prior clips exist
  min_part_secs?: number;
}

export const AI_DEFAULTS: Omit<ModeConfig, 'mode'> = {
  min_duration_secs: 480,
  max_duration_secs: 1200,
  sensitivity_pct: 70,
  skip_music_mins: null,
  force_regenerate: true,
  skip_transcription: false,
  anti_duplicate: { enabled: false, mode: 'subtle' as const },
  skip_regenerate: false,
};

export const LONGFORM_DEFAULTS: Omit<ModeConfig, 'mode'> = {
  min_duration_secs: 480,
  max_duration_secs: 0,
  segment_secs: 600,
  anti_duplicate: { enabled: false, mode: 'subtle' as const },
  skip_regenerate: false,
  min_part_secs: 0,
};

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

// ── PriorClipsResponse ───────────────────────────────────────────────────────
export interface PriorClipsResponse {
  exists: boolean;
  run_id?: number;
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

export interface SSEVideoInfoPayload {
  run_id: number;
  title: string;
  thumbnail_url: string;
  duration_sec: number;
  channel_name?: string;
}

export interface SSEEvent {
  type:
    | 'state_changed'
    | 'gate_opened'
    | 'phase_progress'
    | 'video_info'
    | 'done'
    | 'error'
    | 'cancelled'
    | 'ping';
  data?:
    | SSEStateChangedPayload
    | SSEGateOpenedPayload
    | SSEPhaseProgressPayload
    | SSEVideoInfoPayload
    | { run_id: number; state: RunState; message?: string };
}

// ── UrlHistoryEntry ───────────────────────────────────────────────────────────

export interface UrlHistoryEntry {
  id: number;
  url: string;
  video_title: string | null;
  last_used_at: number;
  use_count: number;
}

// ── Gate payload types (for back-navigation history) ──────────────────────────

export interface UrlPayload {
  kind: 'url';
  url: string;
}

export interface ModePayload {
  kind: 'mode';
  url?: string;
  channel_id?: number;
  mode_config?: ModeConfig;
}

export interface HighlightsPayload {
  kind: 'highlights';
  highlights: HighlightUpdate[];
}

export interface ThumbnailPayload {
  kind: 'thumbnail';
  default_style: string;
  clip_overrides?: ClipThumbnailOverride[];
}

export interface MetadataPayload {
  kind: 'metadata';
  clips: ClipMetadataUpdate[];
}

export interface ClipsPayload {
  kind: 'clips';
  selected_ids: number[];
}

export interface UploadPayload {
  kind: 'upload';
  privacy: 'public' | 'unlisted' | 'private';
}

export type GatePayload =
  | UrlPayload
  | ModePayload
  | HighlightsPayload
  | ThumbnailPayload
  | MetadataPayload
  | ClipsPayload
  | UploadPayload;

export const GATE_STATE_ORDER: RunState[] = [
  'WAITING_URL',
  'WAITING_MODE',
  'WAITING_REVIEW_HIGHLIGHTS',
  'WAITING_THUMBNAIL_CONFIG',
  'WAITING_REVIEW_METADATA',
  'WAITING_REVIEW_CLIPS',
  'WAITING_UPLOAD_CONFIRM',
];

// ── Step rail helpers ─────────────────────────────────────────────────────────

export type StepStatus = 'idle' | 'running' | 'gate' | 'done' | 'error';

export interface StepRailEntry {
  label: string;
  state: RunState;
  status: StepStatus;
}
