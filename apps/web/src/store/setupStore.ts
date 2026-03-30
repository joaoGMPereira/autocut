import { create } from 'zustand';
import type { ToolStatus, ToolInstallState, InstallEvent } from '@/types/setup';
import { createLogger } from '@/lib/logger';

const log = createLogger('setupStore');

interface SetupState {
  tools: ToolStatus[];
  loading: boolean;
  error: string | null;
  installStates: Record<string, ToolInstallState>;
  installLogs: Record<string, string[]>;

  fetchStatus: (goUrl: string) => Promise<void>;
  startInstall: (goUrl: string, toolName: string) => void;
  clearInstallState: (toolName: string) => void;
  allRequiredInstalled: () => boolean;
  missingRequired: () => ToolStatus[];
}

export const useSetupStore = create<SetupState>((set, get) => ({
  tools: [],
  loading: false,
  error: null,
  installStates: {},
  installLogs: {},

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
}));
