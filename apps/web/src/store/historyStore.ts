import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('historyStore');

const LIMIT = 20;

// ─── Types ────────────────────────────────────────────────────────────────────

/** Matches Go's sql.NullInt64 JSON serialization: {"Int64": N, "Valid": bool} */
export interface NullInt64 {
  Int64: number;
  Valid: boolean;
}

export interface RunSummary {
  id: number;
  url: string;
  mode: string;
  status: 'pending' | 'running' | 'done' | 'error';
  /** Unix milliseconds (int64 in Go) */
  started_at: number;
  /** sql.NullInt64 — null when not yet finished */
  finished_at: NullInt64 | null;
  channel_id: NullInt64 | null;
}

export interface RunStep {
  step_name: string;
  status: string;
  /** sql.NullInt64 — null when not yet started */
  started_at: NullInt64 | null;
  /** sql.NullInt64 — null when not yet finished */
  finished_at: NullInt64 | null;
  output_json: unknown;
  error_message: string | null;
}

export interface RunDetail extends RunSummary {
  step_outputs: Record<string, unknown>;
  steps: RunStep[];
}

export interface HistoryFilters {
  channelId: number | null;
  status: string;
  dateFrom: string;
  dateTo: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

/** Resolve a timestamp value from either a plain number or a NullInt64 object. */
function resolveMs(v: number | NullInt64 | null | undefined): number | null {
  if (v == null) return null;
  if (typeof v === 'number') return v;
  return v.Valid ? v.Int64 : null;
}

export function calcDuration(
  started: number | NullInt64 | null,
  finished: number | NullInt64 | null,
): string {
  const startMs = resolveMs(started);
  const finishMs = resolveMs(finished);
  if (startMs == null || finishMs == null) return '—';
  const ms = finishMs - startMs;
  if (ms < 0) return '—';
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  if (m === 0) return `${s}s`;
  return `${m}m ${s}s`;
}

function applyClientFilters(runs: RunSummary[], filters: HistoryFilters): RunSummary[] {
  const fromMs = filters.dateFrom ? new Date(filters.dateFrom).getTime() : null;
  const toMs = filters.dateTo ? new Date(filters.dateTo + 'T23:59:59').getTime() : null;
  return runs.filter((r) => {
    if (filters.status && filters.status !== 'all' && r.status !== filters.status) return false;
    if (fromMs != null && r.started_at < fromMs) return false;
    if (toMs != null && r.started_at > toMs) return false;
    return true;
  });
}

// ─── State interface ──────────────────────────────────────────────────────────

interface HistoryState {
  runs: RunSummary[];
  allRuns: RunSummary[];
  selectedRun: RunDetail | null;
  filters: HistoryFilters;
  page: number;
  totalPages: number;
  loading: boolean;

  fetchRuns: (goUrl: string) => Promise<void>;
  setFilter: (key: keyof HistoryFilters, value: unknown) => void;
  resetFilters: () => void;
  selectRun: (goUrl: string, id: number) => Promise<void>;
  clearSelectedRun: () => void;
  deleteRun: (goUrl: string, id: number) => Promise<void>;
  nextPage: () => void;
  prevPage: () => void;
  exportCSV: () => void;
}

const DEFAULT_FILTERS: HistoryFilters = {
  channelId: null,
  status: 'all',
  dateFrom: '',
  dateTo: '',
};

// ─── Store ────────────────────────────────────────────────────────────────────

export const useHistoryStore = create<HistoryState>((set, get) => ({
  runs: [],
  allRuns: [],
  selectedRun: null,
  filters: { ...DEFAULT_FILTERS },
  page: 1,
  totalPages: 1,
  loading: false,

  fetchRuns: async (goUrl) => {
    const { filters, page } = get();
    log.info('fetching runs', { page, filters });
    set({ loading: true });
    try {
      const params = new URLSearchParams({
        page: String(page),
        limit: String(LIMIT),
      });
      if (filters.channelId != null) {
        params.set('channel_id', String(filters.channelId));
      }
      const res = await fetch(`${goUrl}/api/pipeline/runs?${params.toString()}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { runs: RunSummary[]; total: number };
      const allRuns = data.runs ?? [];
      const filtered = applyClientFilters(allRuns, filters);
      const totalPages = Math.max(1, Math.ceil((data.total ?? 0) / LIMIT));
      log.info('runs fetched', { total: data.total, filtered: filtered.length });
      set({ allRuns, runs: filtered, totalPages, loading: false });
    } catch (err) {
      log.error('fetchRuns failed', { err: err instanceof Error ? err.message : String(err) });
      set({ loading: false });
    }
  },

  setFilter: (key, value) => {
    set((state) => ({
      filters: { ...state.filters, [key]: value },
      page: 1,
    }));
  },

  resetFilters: () => {
    set({ filters: { ...DEFAULT_FILTERS }, page: 1 });
  },

  selectRun: async (goUrl, id) => {
    log.info('selecting run detail', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as {
        run: RunSummary;
        step_outputs: Record<string, unknown>;
        steps: RunStep[];
      };
      const detail: RunDetail = {
        ...data.run,
        step_outputs: data.step_outputs ?? {},
        steps: data.steps ?? [],
      };
      log.info('run detail loaded', { id, steps: detail.steps.length });
      set({ selectedRun: detail });
    } catch (err) {
      log.error('selectRun failed', { id, err: err instanceof Error ? err.message : String(err) });
    }
  },

  clearSelectedRun: () => {
    set({ selectedRun: null });
  },

  deleteRun: async (goUrl, id) => {
    log.info('deleting run', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('run deleted', { id });
      const { selectedRun } = get();
      if (selectedRun?.id === id) set({ selectedRun: null });
      await get().fetchRuns(goUrl);
    } catch (err) {
      log.error('deleteRun failed', { id, err: err instanceof Error ? err.message : String(err) });
    }
  },

  nextPage: () => {
    const { page, totalPages } = get();
    if (page < totalPages) set({ page: page + 1 });
  },

  prevPage: () => {
    const { page } = get();
    if (page > 1) set({ page: page - 1 });
  },

  exportCSV: () => {
    const { runs } = get();
    log.info('exporting CSV', { count: runs.length });
    const headers = ['ID', 'URL', 'Mode', 'Status', 'Started', 'Duration'];
    const rows = runs.map((r) => [
      String(r.id),
      `"${r.url.replace(/"/g, '""')}"`,
      r.mode,
      r.status,
      new Date(r.started_at).toISOString(),
      calcDuration(r.started_at, r.finished_at),
    ]);
    const csv = [headers, ...rows].map((row) => row.join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `pipeline-runs-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  },
}));
