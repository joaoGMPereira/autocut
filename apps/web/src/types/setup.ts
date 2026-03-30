export interface ToolStatus {
  name: string;
  installed: boolean;
  required: boolean;
  path?: string;
  version?: string;
}

export type InstallEventType = 'log' | 'done' | 'error';

export interface InstallEvent {
  type: InstallEventType;
  data: { message?: string; success?: boolean };
}

export type ToolInstallState = 'idle' | 'installing' | 'done' | 'error';

export const AUTO_INSTALL_TOOLS = ['yt-dlp', 'TwitchDownloaderCLI'] as const;
export const PARTIAL_INSTALL_TOOLS = ['whisper'] as const;
export const MANUAL_INSTALL_TOOLS = ['ffmpeg', 'ollama', 'convert'] as const;

export const MANUAL_INSTALL_URLS: Record<string, string> = {
  ffmpeg: 'https://ffmpeg.org/download.html',
  ollama: 'https://ollama.com/download',
  convert: 'https://imagemagick.org/script/download.php',
};

export function isAutoInstallable(toolName: string): boolean {
  return (
    (AUTO_INSTALL_TOOLS as readonly string[]).includes(toolName) ||
    (PARTIAL_INSTALL_TOOLS as readonly string[]).includes(toolName)
  );
}
