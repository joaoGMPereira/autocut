import { create } from 'zustand';
import type { ToolStatus, ToolInstallState, InstallEvent, WhisperModelInfo } from '@/types/setup';
import { createLogger } from '@/lib/logger';

const log = createLogger('setupStore');

interface SetupState {
  tools: ToolStatus[];
  loading: boolean;
  error: string | null;
  installStates: Record<string, ToolInstallState>;
  installLogs: Record<string, string[]>;
  whisperModels: WhisperModelInfo[];
  whisperDownloadStates: Record<string, ToolInstallState>;
  whisperDownloadLogs: Record<string, string[]>;

  fetchStatus: (goUrl: string) => Promise<void>;
  startInstall: (goUrl: string, toolName: string) => void;
  clearInstallState: (toolName: string) => void;
  allRequiredInstalled: () => boolean;
  missingRequired: () => ToolStatus[];
  checkUpdate: (goUrl: string, toolName: string) => Promise<void>;
  fetchWhisperModels: (goUrl: string) => Promise<void>;
  downloadWhisperModel: (goUrl: string, model: string) => void;
}

export const useSetupStore = create<SetupState>((set, get) => ({
  tools: [],
  loading: false,
  error: null,
  installStates: {},
  installLogs: {},
  whisperModels: [],
  whisperDownloadStates: {},
  whisperDownloadLogs: {},

  fetchStatus: async (goUrl: string) => {
    set({ loading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/setup/status`);
      if (!res.ok) throw new Error(`Status check failed: ${res.status}`);
      const data = await res.json();
      log.info('tools status fetched', { count: data.tools?.length });
      set({ tools: data.tools ?? [], loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('status fetch failed', { err: message });
      set({ loading: false, error: message });
    }
  },

  startInstall: (goUrl: string, toolName: string) => {
    log.info('starting install', { tool: toolName });

    set((state) => ({
      installStates: { ...state.installStates, [toolName]: 'installing' },
      installLogs: { ...state.installLogs, [toolName]: [] },
    }));

    // 1. Connect to SSE BEFORE the POST (job_id is deterministic)
    const es = new EventSource(
      `${goUrl}/api/setup/install/${toolName}/stream`,
    );

    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data) as InstallEvent;

        if (event.type === 'log') {
          set((state) => ({
            installLogs: {
              ...state.installLogs,
              [toolName]: [
                ...(state.installLogs[toolName] ?? []),
                event.data.message ?? '',
              ],
            },
          }));
        } else if (event.type === 'done') {
          es.close();
          log.info('install completed', {
            tool: toolName,
            success: event.data.success,
          });
          set((state) => ({
            installStates: {
              ...state.installStates,
              [toolName]: event.data.success ? 'done' : 'error',
            },
          }));
          // Re-fetch status after install
          get().fetchStatus(goUrl);
        } else if (event.type === 'error') {
          es.close();
          log.error('install failed', {
            tool: toolName,
            message: event.data.message,
          });
          set((state) => ({
            installStates: { ...state.installStates, [toolName]: 'error' },
            installLogs: {
              ...state.installLogs,
              [toolName]: [
                ...(state.installLogs[toolName] ?? []),
                `Error: ${event.data.message ?? 'Unknown error'}`,
              ],
            },
          }));
        }
      } catch {
        log.error('failed to parse SSE event', { data: e.data });
      }
    };

    es.onerror = () => {
      es.close();
      set((state) => ({
        installStates: { ...state.installStates, [toolName]: 'error' },
      }));
    };

    // 2. Trigger installation
    fetch(`${goUrl}/api/setup/install/${toolName}`, { method: 'POST' }).catch(
      (err) => {
        log.error('install POST failed', { tool: toolName, err });
      },
    );
  },

  clearInstallState: (toolName: string) => {
    set((state) => ({
      installStates: { ...state.installStates, [toolName]: 'idle' },
      installLogs: { ...state.installLogs, [toolName]: [] },
    }));
  },

  allRequiredInstalled: () => {
    const { tools } = get();
    return (
      tools.length > 0 &&
      tools.filter((t) => t.required).every((t) => t.installed)
    );
  },

  missingRequired: () => {
    const { tools } = get();
    return tools.filter((t) => t.required && !t.installed);
  },

  checkUpdate: async (goUrl: string, toolName: string) => {
    try {
      const res = await fetch(`${goUrl}/api/setup/check-update/${toolName}`);
      if (!res.ok) return;
      const data = await res.json();
      set((state) => ({
        tools: state.tools.map((t) =>
          t.name === toolName
            ? { ...t, latestVersion: data.latest_version, updateAvailable: data.update_available }
            : t,
        ),
      }));
    } catch (err) {
      log.error('check update failed', { tool: toolName, err });
    }
  },

  fetchWhisperModels: async (goUrl: string) => {
    try {
      const res = await fetch(`${goUrl}/api/setup/whisper/models`);
      if (!res.ok) return;
      const data = await res.json();
      set({ whisperModels: data.models ?? [] });
    } catch (err) {
      log.error('fetch whisper models failed', { err });
    }
  },

  downloadWhisperModel: (goUrl: string, model: string) => {
    log.info('starting whisper model download', { model });

    set((state) => ({
      whisperDownloadStates: { ...state.whisperDownloadStates, [model]: 'installing' },
      whisperDownloadLogs: { ...state.whisperDownloadLogs, [model]: [] },
    }));

    const es = new EventSource(
      `${goUrl}/api/setup/whisper/models/${model}/stream`,
    );

    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data) as InstallEvent;

        if (event.type === 'log') {
          set((state) => ({
            whisperDownloadLogs: {
              ...state.whisperDownloadLogs,
              [model]: [
                ...(state.whisperDownloadLogs[model] ?? []),
                event.data.message ?? '',
              ],
            },
          }));
        } else if (event.type === 'done') {
          es.close();
          log.info('whisper model download completed', { model });
          set((state) => ({
            whisperDownloadStates: {
              ...state.whisperDownloadStates,
              [model]: 'done',
            },
          }));
          get().fetchWhisperModels(goUrl);
        } else if (event.type === 'error') {
          es.close();
          log.error('whisper model download failed', { model, message: event.data.message });
          set((state) => ({
            whisperDownloadStates: { ...state.whisperDownloadStates, [model]: 'error' },
            whisperDownloadLogs: {
              ...state.whisperDownloadLogs,
              [model]: [
                ...(state.whisperDownloadLogs[model] ?? []),
                `Error: ${event.data.message ?? 'Unknown error'}`,
              ],
            },
          }));
        }
      } catch {
        log.error('failed to parse whisper SSE event', { data: e.data });
      }
    };

    es.onerror = () => {
      es.close();
      set((state) => ({
        whisperDownloadStates: { ...state.whisperDownloadStates, [model]: 'error' },
      }));
    };

    fetch(`${goUrl}/api/setup/whisper/models/${model}`, { method: 'POST' }).catch(
      (err) => {
        log.error('whisper model POST failed', { model, err });
      },
    );
  },
}));
