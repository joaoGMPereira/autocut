import { create } from 'zustand';
import { createLogger } from '@/lib/logger';
import { useAppStore } from '@/store/appStore';

const log = createLogger('queueStore');

export interface QueueItem {
  id: number;
  clip_id: number;
  channel_id: number;
  status: 'queued' | 'running' | 'done' | 'failed';
  youtube_id?: string;
  youtube_url?: string;
  queue_order: number;
  publish_at?: string;
  error?: string;
  title?: string;
  video_type: string;
  created_at: number;
}

interface QueueStore {
  items: QueueItem[];
  loading: boolean;
  error: string | null;
  fetchQueue: () => Promise<void>;
  retryItem: (id: number) => Promise<void>;
  deleteItem: (id: number) => Promise<void>;
  saveLocal: () => Promise<void>;
  scheduleItem: (id: number, publishAt: string) => Promise<void>;
  bulkSchedule: (startAt: string, intervalMinutes: number) => Promise<void>;
}

export const useQueueStore = create<QueueStore>((set) => ({
  items: [],
  loading: false,
  error: null,

  fetchQueue: async () => {
    const goUrl = useAppStore.getState().goUrl;
    set({ loading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/queue`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as QueueItem[];
      log.info('queue fetched', { count: data.length });
      set({ items: data, loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch queue failed', { err: message });
      set({ loading: false, error: message });
    }
  },

  retryItem: async (id) => {
    const goUrl = useAppStore.getState().goUrl;
    try {
      const res = await fetch(`${goUrl}/api/queue/${id}/retry`, { method: 'POST' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('queue item retried', { id });
      set((state) => ({
        items: state.items.map((item) =>
          item.id === id ? { ...item, status: 'queued', error: undefined } : item
        ),
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('retry item failed', { id, err: message });
      throw err;
    }
  },

  deleteItem: async (id) => {
    const goUrl = useAppStore.getState().goUrl;
    try {
      const res = await fetch(`${goUrl}/api/queue/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('queue item deleted', { id });
      set((state) => ({ items: state.items.filter((item) => item.id !== id) }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('delete item failed', { id, err: message });
      throw err;
    }
  },

  saveLocal: async () => {
    const goUrl = useAppStore.getState().goUrl;
    try {
      const res = await fetch(`${goUrl}/api/queue/save-local`, { method: 'POST' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('queue saved to local');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('save local failed', { err: message });
      throw err;
    }
  },

  scheduleItem: async (id, publishAt) => {
    const goUrl = useAppStore.getState().goUrl;
    try {
      const res = await fetch(`${goUrl}/api/queue/${id}/schedule`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ publish_at: publishAt }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('queue item scheduled', { id, publishAt });
      set((state) => ({
        items: state.items.map((item) =>
          item.id === id ? { ...item, publish_at: publishAt } : item
        ),
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('schedule item failed', { id, err: message });
      throw err;
    }
  },

  bulkSchedule: async (startAt, intervalMinutes) => {
    const goUrl = useAppStore.getState().goUrl;
    try {
      const res = await fetch(`${goUrl}/api/queue/bulk-schedule`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ start_at: startAt, interval_minutes: intervalMinutes }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('bulk schedule applied', { startAt, intervalMinutes });
      // Re-fetch to get updated publish_at values
      const fetchQueue = useQueueStore.getState().fetchQueue;
      await fetchQueue();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('bulk schedule failed', { err: message });
      throw err;
    }
  },
}));
