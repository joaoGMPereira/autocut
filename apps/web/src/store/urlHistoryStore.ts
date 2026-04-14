import { create } from 'zustand';
import { createLogger } from '@/lib/logger';
import type { UrlHistoryEntry } from '@/types/pipeline';

const log = createLogger('urlHistoryStore');

interface UrlHistoryState {
  history: UrlHistoryEntry[];
  isLoading: boolean;
  fetchHistory: (goUrl: string) => Promise<void>;
  removeEntry: (goUrl: string, id: number) => Promise<void>;
  clearAll: (goUrl: string) => Promise<void>;
}

export const useUrlHistoryStore = create<UrlHistoryState>((set) => ({
  history: [],
  isLoading: false,

  fetchHistory: async (goUrl) => {
    set({ isLoading: true });
    try {
      const res = await fetch(`${goUrl}/api/url-history`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { history: UrlHistoryEntry[] };
      set({ history: data.history ?? [], isLoading: false });
    } catch (err) {
      log.error('[urlHistoryStore] fetchHistory failed', { error: err instanceof Error ? err.message : 'Unknown' });
      set({ isLoading: false });
    }
  },

  removeEntry: async (goUrl, id) => {
    set((s) => ({ history: s.history.filter((e) => e.id !== id) }));
    try {
      const res = await fetch(`${goUrl}/api/url-history/${id}`, { method: 'DELETE' });
      if (!res.ok && res.status !== 404) throw new Error(`Failed: ${res.status}`);
    } catch (err) {
      log.error('[urlHistoryStore] removeEntry failed', { id, error: err instanceof Error ? err.message : 'Unknown' });
    }
  },

  clearAll: async (goUrl) => {
    set({ history: [] });
    try {
      const res = await fetch(`${goUrl}/api/url-history`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
    } catch (err) {
      log.error('[urlHistoryStore] clearAll failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  },
}));
