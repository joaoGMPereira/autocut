import { create } from 'zustand';
import { createLogger } from '@/lib/logger';
import type {
  Run,
  Highlight,
  Clip,
  RunState,
  WorkflowMode,
  ModeConfig,
  AdvanceRequest,
  ReviewHighlightsRequest,
  ThumbnailConfigRequest,
  ReviewMetadataRequest,
  ReviewClipsRequest,
  UploadConfirmRequest,
  SSEEvent,
  SSEPhaseProgressPayload,
} from '@/types/pipeline';
import { useDispatcherStore } from '@/store/dispatcherStore';

const log = createLogger('pipelineStore');

// ── Phase progress tracking ───────────────────────────────────────────────────

export interface PhaseProgress {
  phase: string;
  percentDone: number;
  speedKbs?: number;
  etaSec?: number;
}

// ── Store interface ───────────────────────────────────────────────────────────

interface PipelineState {
  // Active run
  activeRunId: number | null;   // kept for backward-compat (equals run?.id ?? null)
  run: Run | null;
  runs: Run[];
  runsTotal: number;
  highlights: Highlight[];
  clips: Clip[];
  phaseProgress: PhaseProgress | null;
  isLoading: boolean;
  error: string | null;
  sseCleanup: (() => void) | null;

  // Actions — run lifecycle
  createRun: (goUrl: string) => Promise<number | null>;
  loadRun: (goUrl: string, id: number) => Promise<void>;
  listRuns: (goUrl: string, limit?: number, offset?: number, channelId?: number) => Promise<void>;
  deleteRun: (goUrl: string, id: number) => Promise<void>;
  cancelRun: (goUrl: string, id: number) => Promise<void>;
  clearRun: () => void;

  // Actions — gate advances
  advance: (goUrl: string, id: number, req: AdvanceRequest) => Promise<void>;
  submitReviewHighlights: (goUrl: string, id: number, req: ReviewHighlightsRequest) => Promise<void>;
  submitThumbnailConfig: (goUrl: string, id: number, req: ThumbnailConfigRequest) => Promise<void>;
  submitReviewMetadata: (goUrl: string, id: number, req: ReviewMetadataRequest) => Promise<void>;
  submitReviewClips: (goUrl: string, id: number, req: ReviewClipsRequest) => Promise<void>;
  submitUploadConfirm: (goUrl: string, id: number, req: UploadConfirmRequest) => Promise<void>;

  // Actions — resource loading
  loadHighlights: (goUrl: string, id: number) => Promise<void>;
  loadClips: (goUrl: string, id: number) => Promise<void>;

  // Actions — SSE
  subscribeSSE: (goUrl: string, runId: number) => () => void;
  stopSSE: () => void;
}

// ── Store ─────────────────────────────────────────────────────────────────────

export const usePipelineStore = create<PipelineState>((set, get) => ({
  activeRunId: null,
  run: null,
  runs: [],
  runsTotal: 0,
  highlights: [],
  clips: [],
  phaseProgress: null,
  isLoading: false,
  error: null,
  sseCleanup: null,

  // ── Run lifecycle ──────────────────────────────────────────────────────────

  createRun: async (goUrl) => {
    log.info('[pipelineStore] creating new run');
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs`, { method: 'POST' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { run_id: number; state: RunState };
      log.info('[pipelineStore] run created', { run_id: data.run_id });
      // Load the full run
      const runRes = await fetch(`${goUrl}/api/pipeline/runs/${data.run_id}`);
      if (!runRes.ok) throw new Error(`Failed to load run: ${runRes.status}`);
      const runData = (await runRes.json()) as { run: Run };
      set({ run: runData.run, activeRunId: runData.run.id, isLoading: false });
      return data.run_id;
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to create run';
      log.error('[pipelineStore] createRun failed', { error: msg });
      set({ isLoading: false, error: msg });
      return null;
    }
  },

  loadRun: async (goUrl, id) => {
    log.info('[pipelineStore] loading run', { id });
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { run: Run };
      set({ run: data.run, activeRunId: data.run.id, isLoading: false });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to load run';
      log.error('[pipelineStore] loadRun failed', { error: msg });
      set({ isLoading: false, error: msg });
    }
  },

  listRuns: async (goUrl, limit = 20, offset = 0, channelId) => {
    log.info('[pipelineStore] listing runs');
    set({ isLoading: true, error: null });
    try {
      const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
      if (channelId != null) params.set('channel_id', String(channelId));
      const res = await fetch(`${goUrl}/api/pipeline/runs?${params}`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { runs: Run[]; total: number };
      set({ runs: data.runs ?? [], runsTotal: data.total ?? 0, isLoading: false });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to list runs';
      log.error('[pipelineStore] listRuns failed', { error: msg });
      set({ isLoading: false, error: msg });
    }
  },

  deleteRun: async (goUrl, id) => {
    log.info('[pipelineStore] deleting run', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const { run } = get();
      if (run?.id === id) set({ run: null, activeRunId: null });
      set((s) => ({ runs: s.runs.filter((r) => r.id !== id) }));
    } catch (err) {
      log.error('[pipelineStore] deleteRun failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  },

  cancelRun: async (goUrl, id) => {
    log.info('[pipelineStore] cancelling run', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/cancel`, { method: 'POST' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState };
      set((s) => ({
        run: s.run?.id === id ? { ...s.run, state: data.state } : s.run,
      }));
    } catch (err) {
      log.error('[pipelineStore] cancelRun failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  },

  clearRun: () => {
    get().stopSSE();
    set({ run: null, activeRunId: null, highlights: [], clips: [], phaseProgress: null, error: null });
  },

  // ── Gate advances ──────────────────────────────────────────────────────────

  advance: async (goUrl, id, req) => {
    log.info('[pipelineStore] advance', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/advance`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState };
      set((s) => ({ run: s.run?.id === id ? { ...s.run, state: data.state } : s.run }));
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'advance failed';
      log.error('[pipelineStore] advance failed', { error: msg });
      set({ error: msg });
    }
  },

  submitReviewHighlights: async (goUrl, id, req) => {
    log.info('[pipelineStore] submitReviewHighlights', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/gates/review-highlights`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState };
      set((s) => ({ run: s.run?.id === id ? { ...s.run, state: data.state } : s.run }));
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'submitReviewHighlights failed';
      log.error('[pipelineStore] submitReviewHighlights failed', { error: msg });
      set({ error: msg });
    }
  },

  submitThumbnailConfig: async (goUrl, id, req) => {
    log.info('[pipelineStore] submitThumbnailConfig', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/gates/thumbnail-config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState };
      set((s) => ({ run: s.run?.id === id ? { ...s.run, state: data.state } : s.run }));
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'submitThumbnailConfig failed';
      log.error('[pipelineStore] submitThumbnailConfig failed', { error: msg });
      set({ error: msg });
    }
  },

  submitReviewMetadata: async (goUrl, id, req) => {
    log.info('[pipelineStore] submitReviewMetadata', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/gates/review-metadata`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState };
      set((s) => ({ run: s.run?.id === id ? { ...s.run, state: data.state } : s.run }));
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'submitReviewMetadata failed';
      log.error('[pipelineStore] submitReviewMetadata failed', { error: msg });
      set({ error: msg });
    }
  },

  submitReviewClips: async (goUrl, id, req) => {
    log.info('[pipelineStore] submitReviewClips', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/gates/review-clips`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState };
      set((s) => ({ run: s.run?.id === id ? { ...s.run, state: data.state } : s.run }));
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'submitReviewClips failed';
      log.error('[pipelineStore] submitReviewClips failed', { error: msg });
      set({ error: msg });
    }
  },

  submitUploadConfirm: async (goUrl, id, req) => {
    log.info('[pipelineStore] submitUploadConfirm', { id });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/gates/upload-confirm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState };
      set((s) => ({ run: s.run?.id === id ? { ...s.run, state: data.state } : s.run }));
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'submitUploadConfirm failed';
      log.error('[pipelineStore] submitUploadConfirm failed', { error: msg });
      set({ error: msg });
    }
  },

  // ── Resource loading ───────────────────────────────────────────────────────

  loadHighlights: async (goUrl, id) => {
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/highlights`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { highlights: Highlight[] };
      set({ highlights: data.highlights ?? [] });
    } catch (err) {
      log.error('[pipelineStore] loadHighlights failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  },

  loadClips: async (goUrl, id) => {
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/clips`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { clips: Clip[] };
      set({ clips: data.clips ?? [] });
    } catch (err) {
      log.error('[pipelineStore] loadClips failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  },

  // ── SSE ────────────────────────────────────────────────────────────────────

  subscribeSSE: (goUrl, runId) => {
    log.info('[pipelineStore] subscribing SSE', { runId });
    const es = new EventSource(`${goUrl}/api/pipeline/runs/${runId}/stream`);

    es.onmessage = (e: MessageEvent) => {
      try {
        const evt = JSON.parse(e.data as string) as SSEEvent;
        if (evt.type === 'ping') return;

        if (evt.type === 'state_changed' || evt.type === 'gate_opened') {
          const payload = evt.data as { run_id: number; state: RunState };
          set((s) => ({
            run: s.run?.id === payload.run_id ? { ...s.run, state: payload.state } : s.run,
          }));
        }

        if (evt.type === 'phase_progress') {
          const payload = evt.data as SSEPhaseProgressPayload;
          set({
            phaseProgress: {
              phase: payload.phase,
              percentDone: payload.percent_done,
              speedKbs: payload.speed_kbs,
              etaSec: payload.eta_sec,
            },
          });
          if (payload.warning) {
            useDispatcherStore.getState().markError(
              `tool_error:logo_watermark:${payload.run_id}`,
              payload.warning,
            );
          }
        }

        if (evt.type === 'done') {
          es.close();
          const payload = evt.data as { run_id: number; state: RunState };
          set((s) => ({
            run: s.run?.id === payload.run_id ? { ...s.run, state: 'DONE' } : s.run,
            sseCleanup: null,
            phaseProgress: null,
          }));
        }

        if (evt.type === 'error') {
          es.close();
          const payload = evt.data as { run_id: number; state: RunState; message?: string };
          set((s) => ({
            run: s.run?.id === payload.run_id ? { ...s.run, state: 'ERROR', error: payload.message ?? '' } : s.run,
            sseCleanup: null,
            phaseProgress: null,
          }));
        }

        if (evt.type === 'cancelled') {
          es.close();
          const payload = evt.data as { run_id: number; state: RunState };
          set((s) => ({
            run: s.run?.id === payload.run_id ? { ...s.run, state: 'CANCELLED' } : s.run,
            sseCleanup: null,
            phaseProgress: null,
          }));
        }
      } catch {
        log.error('[pipelineStore] failed to parse SSE event', { runId });
      }
    };

    es.onerror = () => {
      es.close();
      log.error('[pipelineStore] SSE connection error', { runId });
      set({ sseCleanup: null, phaseProgress: null });
    };

    const cleanup = () => { try { es.close(); } catch { /* ignore */ } };
    set({ sseCleanup: cleanup });
    return cleanup;
  },

  stopSSE: () => {
    const { sseCleanup } = get();
    if (sseCleanup) {
      sseCleanup();
      set({ sseCleanup: null });
      log.info('[pipelineStore] SSE stopped');
    }
  },
}));

// ── Derived helpers ───────────────────────────────────────────────────────────

export function isGateState(state: RunState): boolean {
  return [
    'WAITING_URL',
    'WAITING_MODE',
    'WAITING_REVIEW_HIGHLIGHTS',
    'WAITING_THUMBNAIL_CONFIG',
    'WAITING_REVIEW_METADATA',
    'WAITING_REVIEW_CLIPS',
    'WAITING_UPLOAD_CONFIRM',
  ].includes(state);
}

export function isComputeState(state: RunState): boolean {
  return ['EXECUTING', 'GENERATING_CLIPS', 'UPLOADING'].includes(state);
}

export function isTerminalState(state: RunState): boolean {
  return ['DONE', 'ERROR', 'CANCELLED'].includes(state);
}

export type { Run, Highlight, Clip, RunState, WorkflowMode, ModeConfig };
