// ─── Processing Status ────────────────────────────────────────────────────────

export type ProcessingStatus = 'idle' | 'running' | 'done' | 'error';

// ─── Update Progress ──────────────────────────────────────────────────────────

export type UpdateProgressStatus =
  | 'idle'
  | 'checking'
  | 'available'
  | 'downloading'
  | 'downloaded'
  | 'not-available'
  | 'error';

export interface UpdateProgress {
  status: UpdateProgressStatus;
  updateInfo?: UpdateInfo;
  percent?: number;
  bytesPerSecond?: number;
  transferred?: number;
  total?: number;
  error?: string;
}

export interface UpdateInfo {
  version: string;
  releaseDate: string;
  releaseNotes?: string;
}

// ─── Jobs ─────────────────────────────────────────────────────────────────────

export interface DownloadJob {
  id: string;
  url: string;
  type: 'youtube' | 'twitch';
  status: ProcessingStatus;
  progress?: number;
  outputPath?: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ClipJob {
  id: string;
  sourceVideoId: string;
  timestamps: Array<{ start: number; end: number; label?: string }>;
  status: ProcessingStatus;
  progress?: number;
  outputPaths?: string[];
  error?: string;
  createdAt: string;
  updatedAt: string;
}

export interface UploadJob {
  id: string;
  videoPath: string;
  channelId: string;
  title: string;
  description: string;
  tags?: string[];
  scheduledAt?: string;
  status: ProcessingStatus;
  progress?: number;
  youtubeVideoId?: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

// ─── Channel ──────────────────────────────────────────────────────────────────

export interface Channel {
  id: string;
  name: string;
  youtubeChannelId: string;
  handle?: string;
  thumbnailUrl?: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

// ─── Video Meta ───────────────────────────────────────────────────────────────

export interface VideoMeta {
  id: string;
  url: string;
  title: string;
  description?: string;
  duration: number;
  thumbnailUrl?: string;
  channelName?: string;
  uploadDate?: string;
  viewCount?: number;
  localPath?: string;
}

// ─── SSE Events ───────────────────────────────────────────────────────────────

export interface SSEProgressEvent {
  jobId: string;
  status: ProcessingStatus;
  progress?: number;
  message?: string;
  error?: string;
}
