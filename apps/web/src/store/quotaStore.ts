import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('quotaStore');

export interface QuotaHistory {
  date: string;
  units_used: number;
  upload_count: number;
}

export interface QuotaBreakdown {
  uploads_used: number;
  uploads_limit: number;
  updates_used: number;
  updates_limit: number;
  reads_used: number;
  reads_limit: number;
  total_units: number;
  total_limit: number;
}

interface QuotaStore {
  breakdown: QuotaBreakdown | null;
  history: QuotaHistory[];
  loading: boolean;
  fetchBreakdown: () => Promise<void>;
  fetchHistory: () => Promise<void>;
  resetQuota: () => Promise<void>;
}

const goUrl = (): string =>
  process.env.NEXT_PUBLIC_GO_URL ?? 'http://127.0.0.1:4071';

export const useQuotaStore = create<QuotaStore>((set) => ({
  breakdown: null,
  history: [],
  loading: false,

  fetchBreakdown: async () => {
    set({ loading: true });
    try {
      const res = await fetch(`${goUrl()}/api/stats`);
      if (!res.ok) {
        throw new Error(`stats fetch failed: ${res.status}`);
      }
      const data = (await res.json()) as { quota_breakdown: QuotaBreakdown };
      log.info('quota breakdown loaded', { total: data.quota_breakdown?.total_units });
      set({ breakdown: data.quota_breakdown ?? null, loading: false });
    } catch (err) {
      log.error('fetchBreakdown failed', { err });
      set({ loading: false });
    }
  },

  fetchHistory: async () => {
    set({ loading: true });
    try {
      const res = await fetch(`${goUrl()}/api/quota/history`);
      if (!res.ok) {
        throw new Error(`quota history fetch failed: ${res.status}`);
      }
      const data = (await res.json()) as { history: QuotaHistory[] };
      log.info('quota history loaded', { days: data.history?.length });
      set({ history: data.history ?? [], loading: false });
    } catch (err) {
      log.error('fetchHistory failed', { err });
      set({ loading: false });
    }
  },

  resetQuota: async () => {
    try {
      const res = await fetch(`${goUrl()}/api/quota/reset`, { method: 'POST' });
      if (!res.ok) {
        throw new Error(`quota reset failed: ${res.status}`);
      }
      log.info('quota reset successful');
    } catch (err) {
      log.error('resetQuota failed', { err });
    }
  },
}));
