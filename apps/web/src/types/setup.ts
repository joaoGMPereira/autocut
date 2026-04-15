export interface ToolStatus {
  name: string;
  installed: boolean;
  required: boolean;
  path?: string;
  version?: string;
  latestVersion?: string;
  updateAvailable?: boolean;
  versionOk?: boolean;   // undefined = no min version configured
  source?: 'path' | 'custom' | 'autocut_bin' | 'no_model';
}

export interface HardwareProfile {
  total_ram_mb: number;
  cpu_cores: number;
  arch: 'arm64' | 'amd64' | string;
  gpu_vendor: 'apple_metal' | 'nvidia' | 'amd' | 'none' | 'unknown';
  gpu_vram_mb: number;
}

export interface ModelRecommendation {
  whisper_tier: 'tiny' | 'base' | 'small' | 'medium' | 'large';
  whisper_reason: string;
  whisper_size_mb: number;
  ollama_model: string;
  ollama_reason: string;
  upgrade_available: boolean;
}

export interface WhisperModelInfo {
  name: string;
  size_mb: number;
  downloaded: boolean;
  path?: string;
  active?: boolean;
}

export type InstallEventType = 'log' | 'done' | 'error';

export interface InstallEvent {
  type: InstallEventType;
  data: { message?: string; success?: boolean };
}

export type ToolInstallState = 'idle' | 'installing' | 'done' | 'error';

export const AUTO_INSTALL_TOOLS = ['yt-dlp', 'TwitchDownloaderCLI'] as const;
export const PARTIAL_INSTALL_TOOLS = ['whisper'] as const;
export const MANUAL_INSTALL_TOOLS = ['ffmpeg', 'ollama', 'convert', 'brew'] as const;

export const MANUAL_INSTALL_URLS: Record<string, string> = {
  ffmpeg: 'https://ffmpeg.org/download.html',
  ollama: 'https://ollama.com/download',
  convert: 'https://imagemagick.org/script/download.php',
  brew: 'https://brew.sh',
};

export function isAutoInstallable(toolName: string): boolean {
  return (
    (AUTO_INSTALL_TOOLS as readonly string[]).includes(toolName) ||
    (PARTIAL_INSTALL_TOOLS as readonly string[]).includes(toolName)
  );
}
