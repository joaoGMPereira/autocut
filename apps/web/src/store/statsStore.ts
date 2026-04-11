import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('statsStore');

export interface PipelineRunSummary {
  id: number;
  status: string;
  mode: string;
  started_at: number;
  finished_at: number | null;
}

export interface StatsData {
  active_downloads: number;
  pending_uploads: number;
  total_clips: number;
  total_shorts: number;
  channels_count: number;
  quota_used_today: number;
  recent_pipeline_runs: PipelineRunSummary[];
}

interface StatsState {
  stats: StatsData | null;
  loading: boolean;
  fetchStats: () => Promise<void>;
}

export const useStatsStore = create<StatsState>((set) => ({
  stats: null,
  loading: false,

  fetchStats: async () => {
    set({ loading: true });
    try {
      const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://127.0.0.1:4071';
      const res = await fetch(`${goUrl}/api/stats`);
      if (!res.ok) {
        throw new Error(`stats fetch failed: ${res.status}`);
      }
      const data = (await res.json()) as StatsData;
      log.info('stats loaded', { active: data.active_downloads, clips: data.total_clips });
      set({ stats: data, loading: false });
    } catch (err) {
      log.error('fetchStats failed', { err });
      set({ loading: false });
    }
  },
}));
