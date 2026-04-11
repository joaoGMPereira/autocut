// Types for the standalone download handler SSE stream.
// Endpoint: GET /api/download/{job_id}/stream

export interface SSEVideoInfoPayload {
  job_id: string;
  title: string;
  thumbnail_url: string;
  duration_sec: number;
}

export interface SSEDownloadDonePayload {
  job_id: string;
}

export interface SSEDownloadErrorPayload {
  job_id: string;
  message: string;
}

export type DownloadSSEEvent =
  | { type: 'video_info'; data: SSEVideoInfoPayload }
  | { type: 'done'; data: SSEDownloadDonePayload }
  | { type: 'error'; data: SSEDownloadErrorPayload }
  | { type: 'ping'; data?: unknown };

// Parsed, camelCase representation used in component state.
export interface VideoInfoData {
  title: string;
  thumbnailUrl: string;
  durationSec: number;
}
