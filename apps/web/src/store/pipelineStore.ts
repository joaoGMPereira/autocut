import { create } from 'zustand';
import { createLogger } from '@/lib/logger';
import type { ModeConfig } from '@/types/pipeline';
import { hydrateDispatcherFromRun } from '@/lib/backgroundDownload';

const log = createLogger('pipelineStore');

export interface PipelineRunStep {
  id: number;
  step: string;
  status: string;
  jobId: string;
  error: string;
  startedAt: number | null;
  finishedAt: number | null;
}

export interface PipelineRun {
  id: number;
  url: string;
  mode: string;
  channelId: number | null;
  status: string;
  progress: number;
  currentStep: string;
  stepOutputs: Record<string, unknown>;
  steps: PipelineRunStep[];
  error: string;
  startedAt: number;
  finishedAt: number | null;
}

interface PipelineState {
  activeRunId: number | null;
  run: PipelineRun | null;
  runs: PipelineRun[];
  runsTotal: number;
  isLoading: boolean;
  error: string | null;
  createRun: (goUrl: string, url: string, mode?: string, channelId?: number, config?: ModeConfig) => Promise<number>;
  patchRunMode: (goUrl: string, id: number, config: ModeConfig) => Promise<void>;
  loadRun: (goUrl: string, id: number) => Promise<void>;
  refreshRun: (goUrl: string) => Promise<void>;
  listRuns: (goUrl: string, limit?: number, offset?: number, channelId?: number) => Promise<void>;
  deleteRun: (goUrl: string, id: number) => Promise<void>;
  clearRun: () => void;
}

export const usePipelineStore = create<PipelineState>((set, get) => ({
  activeRunId: null,
  run: null,
  runs: [],
  runsTotal: 0,
  isLoading: false,
  error: null,

  createRun: async (goUrl, url, mode = 'manual', channelId, config) => {
    log.info('creating pipeline run', { url, mode });
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, mode, channel_id: channelId, config: config ?? null }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const { run_id } = (await res.json()) as { run_id: number };
      log.info('pipeline run created', { id: run_id });
      // Load the full run from the server
      const runRes = await fetch(`${goUrl}/api/pipeline/runs/${run_id}`);
      if (!runRes.ok) throw new Error(`Failed to load run: ${runRes.status}`);
      const runData = (await runRes.json()) as { run: PipelineRun; step_outputs: Record<string, unknown>; steps: PipelineRunStep[] };
      const fullRun: PipelineRun = {
        ...runData.run,
        stepOutputs: runData.step_outputs ?? {},
        steps: runData.steps ?? [],
      };
      set({ activeRunId: run_id, run: fullRun, isLoading: false });
      return run_id;
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to create run';
      log.error('createRun failed', { error: msg });
      set({ isLoading: false, error: msg });
      throw err;
    }
  },

  patchRunMode: async (goUrl, id, config) => {
    log.info('patching run mode', { id, mode: config.mode });
    const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/mode`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode: config.mode, config }),
    });
    if (!res.ok) throw new Error(`Failed to patch run mode: ${res.status}`);
  },

  loadRun: async (goUrl, id) => {
    log.info('loading pipeline run', { id });
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { run: PipelineRun; step_outputs: Record<string, unknown>; steps: PipelineRunStep[] };
      const fullRun: PipelineRun = {
        ...data.run,
        stepOutputs: data.step_outputs ?? {},
        steps: data.steps ?? [],
      };
      set({ activeRunId: fullRun.id, run: fullRun, isLoading: false });
      hydrateDispatcherFromRun(fullRun);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to load run';
      log.error('loadRun failed', { error: msg });
      set({ isLoading: false, error: msg });
    }
  },

  refreshRun: async (goUrl) => {
    const { activeRunId } = get();
    if (activeRunId == null) return;
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${activeRunId}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { run: PipelineRun; step_outputs: Record<string, unknown>; steps: PipelineRunStep[] };
      const fullRun: PipelineRun = {
        ...data.run,
        stepOutputs: data.step_outputs ?? {},
        steps: data.steps ?? [],
      };
      set({ run: fullRun });
    } catch (err) {
      log.error('refreshRun failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  },

  listRuns: async (goUrl, limit = 20, offset = 0, channelId) => {
    log.info('listing pipeline runs', { limit, offset, channelId });
    set({ isLoading: true, error: null });
    try {
      const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
      if (channelId != null) params.set('channel_id', String(channelId));
      const res = await fetch(`${goUrl}/api/pipeline/runs?${params}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { runs: PipelineRun[]; total: number };
      set({ runs: data.runs, runsTotal: data.total, isLoading: false });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to list runs';
      log.error('listRuns failed', { error: msg });
      set({ isLoading: false, error: msg });
    }
  },

  deleteRun: async (goUrl, id) => {
    log.info('deleting pipeline run', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const { activeRunId } = get();
      if (activeRunId === id) {
        set({ activeRunId: null, run: null });
      }
      set((state) => ({ runs: state.runs.filter((r) => r.id !== id) }));
      log.info('pipeline run deleted', { id });
    } catch (err) {
      log.error('deleteRun failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  },

  clearRun: () => {
    set({ activeRunId: null, run: null, error: null });
  },
}));
