import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('massUpdateStore');

// ─── Types ────────────────────────────────────────────────────────────────────

export interface RemoteVideo {
  id: number;
  youtube_id: string;
  title: string;
  description: string;
  tags: string; // comma-separated JSON
  category_id: number;
  published_at: string;
}

export interface MassUpdateFilter {
  date_from?: string;
  date_to?: string;
  category_id?: number;
}

export interface MassUpdateUpdates {
  title_template?: string; // placeholders: {original}, {channel}, {date}
  description_append?: string;
  tags_add?: string[];
  tags_remove?: string[];
  hashtags?: string[];
  category_id?: number;
}

interface DryRunPreviewItem {
  youtube_id: string;
  old_title: string;
  new_title: string;
}

interface MassUpdateStore {
  channelId: number | null;
  videos: RemoteVideo[];
  totalVideos: number;
  currentPage: number;
  filter: MassUpdateFilter;
  updates: MassUpdateUpdates;
  dryRunPreview: DryRunPreviewItem[] | null;
  jobId: string | null;
  progress: { done: number; total: number } | null;
  status: 'idle' | 'loading' | 'dry_run_done' | 'running' | 'done' | 'error';
  error: string | null;

  setChannelId: (id: number) => void;
  setFilter: (f: Partial<MassUpdateFilter>) => void;
  setUpdates: (u: Partial<MassUpdateUpdates>) => void;
  fetchVideos: (goUrl: string, page?: number) => Promise<void>;
  runDryRun: (goUrl: string) => Promise<void>;
  runUpdate: (goUrl: string) => Promise<void>;
  subscribeProgress: (goUrl: string) => () => void;
  reset: () => void;
}

// ─── Initial State ────────────────────────────────────────────────────────────

const initialState = {
  channelId: null,
  videos: [],
  totalVideos: 0,
  currentPage: 1,
  filter: {} as MassUpdateFilter,
  updates: {} as MassUpdateUpdates,
  dryRunPreview: null,
  jobId: null,
  progress: null,
  status: 'idle' as const,
  error: null,
};

// ─── Store ────────────────────────────────────────────────────────────────────

export const useMassUpdateStore = create<MassUpdateStore>((set, get) => ({
  ...initialState,

  setChannelId: (id) => {
    set({ channelId: id, videos: [], totalVideos: 0, currentPage: 1, dryRunPreview: null });
  },

  setFilter: (f) => {
    set((state) => ({ filter: { ...state.filter, ...f } }));
  },

  setUpdates: (u) => {
    set((state) => ({ updates: { ...state.updates, ...u } }));
  },

  fetchVideos: async (goUrl, page = 1) => {
    const { channelId, filter } = get();
    if (channelId === null) return;

    set({ status: 'loading', error: null });
    try {
      const params = new URLSearchParams({ channel_id: String(channelId), page: String(page) });
      if (filter.date_from) params.set('date_from', filter.date_from);
      if (filter.date_to) params.set('date_to', filter.date_to);
      if (filter.category_id !== undefined) params.set('category_id', String(filter.category_id));

      const res = await fetch(`${goUrl}/api/videos/remote?${params.toString()}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);

      const data = (await res.json()) as { videos: RemoteVideo[]; page: number; total: number; cached: boolean };
      log.info('remote videos fetched', { count: data.videos.length, total: data.total, cached: data.cached });
      set({ videos: data.videos, totalVideos: data.total, currentPage: data.page, status: 'idle' });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch remote videos failed', { err: message });
      set({ status: 'error', error: message });
    }
  },

  runDryRun: async (goUrl) => {
    const { channelId, filter, updates } = get();
    if (channelId === null) return;

    set({ status: 'loading', error: null, dryRunPreview: null });
    try {
      const res = await fetch(`${goUrl}/api/videos/mass-update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ channel_id: channelId, filter, updates, dry_run: true }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);

      const data = (await res.json()) as { dry_run: true; preview: DryRunPreviewItem[]; total: number };
      log.info('dry run complete', { affected: data.total });
      set({ status: 'dry_run_done', dryRunPreview: data.preview });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('dry run failed', { err: message });
      set({ status: 'error', error: message });
    }
  },

  runUpdate: async (goUrl) => {
    const { channelId, filter, updates } = get();
    if (channelId === null) return;

    set({ status: 'running', error: null, progress: null, jobId: null });
    try {
      const res = await fetch(`${goUrl}/api/videos/mass-update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ channel_id: channelId, filter, updates }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);

      const data = (await res.json()) as { job_id: string };
      log.info('mass update job started', { jobId: data.job_id });
      set({ jobId: data.job_id });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('mass update failed', { err: message });
      set({ status: 'error', error: message });
    }
  },

  subscribeProgress: (goUrl) => {
    const { jobId } = get();
    if (!jobId) return () => undefined;

    const url = `${goUrl}/api/videos/mass-update/${jobId}/stream`;
    log.info('subscribing to progress SSE', { jobId, url });
    const es = new EventSource(url);

    es.onmessage = (event) => {
      try {
        type MassUpdateEvent =
          | { type: 'progress'; index: number; total: number; youtube_id: string; new_title: string }
          | { type: 'done'; total: number };

        const payload = JSON.parse(event.data as string) as MassUpdateEvent;

        if (payload.type === 'progress') {
          set({ progress: { done: payload.index, total: payload.total } });
        } else if (payload.type === 'done') {
          log.info('mass update done', { total: payload.total });
          set({
            status: 'done',
            progress: { done: payload.total, total: payload.total },
          });
          es.close();
        }
      } catch (err) {
        log.error('SSE parse error', { err: err instanceof Error ? err.message : String(err) });
      }
    };

    es.onerror = () => {
      log.error('SSE connection error', { jobId });
      set({ status: 'error', error: 'Progress stream connection lost' });
      es.close();
    };

    return () => {
      log.debug('unsubscribing progress SSE', { jobId });
      es.close();
    };
  },

  reset: () => {
    set(initialState);
  },
}));
