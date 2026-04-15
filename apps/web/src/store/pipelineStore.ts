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
  SSEVideoInfoPayload,
  SSEDownloadCompletePayload,
  SSEPreviewProgressPayload,
  SSEPreviewReadyPayload,
  SSEPreviewErrorPayload,
  GatePayload,
} from '@/types/pipeline';
import { GATE_STATE_ORDER } from '@/types/pipeline';
import { useDispatcherStore } from '@/store/dispatcherStore';

const log = createLogger('pipelineStore');

// ── Phase progress tracking ───────────────────────────────────────────────────

export interface PhaseProgress {
  phase: string;
  percentDone: number;
  speedKbs?: number;
  etaSec?: number;
}

export interface VideoInfo {
  title: string;
  thumbnailUrl: string;
  durationSec: number;
  channelName?: string;
}

// ── Store interface ───────────────────────────────────────────────────────────

export type PreviewStatus = 'idle' | 'generating' | 'ready' | 'error';

interface PipelineState {
  // Active run
  activeRunId: number | null;   // kept for backward-compat (equals run?.id ?? null)
  run: Run | null;
  runs: Run[];
  runsTotal: number;
  highlights: Highlight[];
  clips: Clip[];
  phaseProgress: PhaseProgress | null;
  videoInfo: VideoInfo | null;
  isLoading: boolean;
  error: string | null;
  sseCleanup: (() => void) | null;

  // Download state (parallel download while on Mode screen)
  downloadComplete: boolean;
  videoReused: boolean;
  pendingReuse: boolean;

  // Preview state
  previewStatus: PreviewStatus;
  previewPercent: number;
  previewUrl: string | null;
  previewError: string | null;

  // Navigation history
  navHistory: RunState[];
  navIndex: number;
  gateHistory: Partial<Record<RunState, GatePayload>>;

  // Computed accessors
  displayedState: () => RunState;
  isNavigatedBack: () => boolean;

  // Actions — run lifecycle
  createRun: (goUrl: string) => Promise<number | null>;
  loadRun: (goUrl: string, id: number) => Promise<void>;
  listRuns: (goUrl: string, limit?: number, offset?: number, channelId?: number) => Promise<void>;
  deleteRun: (goUrl: string, id: number) => Promise<void>;
  cancelRun: (goUrl: string, id: number) => Promise<void>;
  clearRun: () => void;
  confirmReuse: () => void;

  // Actions — gate advances
  advance: (goUrl: string, id: number, req: AdvanceRequest) => Promise<void>;
  submitReviewHighlights: (goUrl: string, id: number, req: ReviewHighlightsRequest) => Promise<void>;
  submitThumbnailConfig: (goUrl: string, id: number, req: ThumbnailConfigRequest) => Promise<void>;
  submitReviewMetadata: (goUrl: string, id: number, req: ReviewMetadataRequest) => Promise<void>;
  submitReviewClips: (goUrl: string, id: number, req: ReviewClipsRequest) => Promise<void>;
  submitUploadConfirm: (goUrl: string, id: number, req: UploadConfirmRequest) => Promise<void>;

  // Actions — navigation
  navigateBack: () => void;
  navigateForward: () => void;
  recordGateSubmission: (state: RunState, payload: GatePayload) => void;
  resubmitFromBacked: (goUrl: string, targetGate: RunState, newPayload: GatePayload) => Promise<void>;

  // Actions — resource loading
  loadHighlights: (goUrl: string, id: number) => Promise<void>;
  loadClips: (goUrl: string, id: number) => Promise<void>;

  // Actions — preview
  generatePreview: (goUrl: string, runId: number, modeConfig: ModeConfig) => Promise<void>;
  redownload: (goUrl: string, runId: number) => Promise<void>;

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
  videoInfo: null,
  isLoading: false,
  error: null,
  sseCleanup: null,
  downloadComplete: false,
  videoReused: false,
  pendingReuse: false,

  // Preview state
  previewStatus: 'idle',
  previewPercent: 0,
  previewUrl: null,
  previewError: null,

  navHistory: [],
  navIndex: 0,
  gateHistory: {},

  displayedState: () => {
    const { navHistory, navIndex } = get();
    return navHistory[navIndex] ?? 'WAITING_URL';
  },

  isNavigatedBack: () => {
    const { navHistory, navIndex } = get();
    return navIndex < navHistory.length - 1;
  },

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
      set({ run: runData.run, activeRunId: runData.run.id, isLoading: false,
            navHistory: ['WAITING_URL'], navIndex: 0 });
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
    set({
      run: null,
      activeRunId: null,
      highlights: [],
      clips: [],
      phaseProgress: null,
      videoInfo: null,
      error: null,
      previewStatus: 'idle',
      previewPercent: 0,
      previewUrl: null,
      previewError: null,
      navHistory: [],
      navIndex: 0,
      gateHistory: {},
      downloadComplete: false,
      videoReused: false,
      pendingReuse: false,
    });
  },

  confirmReuse: () => {
    const { run } = get();
    if (!run) return;
    set((s) => {
      const newNavHistory = [...s.navHistory, run.state as RunState];
      return {
        pendingReuse: false,
        navHistory: newNavHistory,
        navIndex: newNavHistory.length - 1,
      };
    });
  },

  // ── Gate advances ──────────────────────────────────────────────────────────

  advance: async (goUrl, id, req) => {
    log.info('[pipelineStore] advance', { id });
    // Record gate submission based on current run state
    const currentState = get().run?.state ?? 'WAITING_URL';
    if (currentState === 'WAITING_URL' && req.url) {
      get().recordGateSubmission('WAITING_URL', { kind: 'url', url: req.url });
      console.log('[pipelineStore] recordGateSubmission', 'WAITING_URL', { kind: 'url', url: req.url });
    } else if (currentState === 'WAITING_MODE') {
      const payload: GatePayload = { kind: 'mode', url: req.url, channel_id: req.channel_id, mode_config: req.mode_config };
      get().recordGateSubmission('WAITING_MODE', payload);
      console.log('[pipelineStore] recordGateSubmission', 'WAITING_MODE', payload);
    }
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${id}/advance`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { state: RunState; video_reused?: boolean };
      const isReused = data.video_reused ?? false;
      set((s) => {
        if (isReused) {
          // Reuse detected — block auto-navigation, let StepUrl show reuse card
          return {
            run: s.run?.id === id ? { ...s.run, state: data.state } : s.run,
            videoReused: true,
            downloadComplete: true,
            pendingReuse: true,
          };
        }
        // Normal case — navigate to the new state directly from HTTP response
        // (SSE state_changed may not arrive if EventSource connects after Advance returns)
        const newNavHistory = [...s.navHistory, data.state];
        return {
          run: s.run?.id === id ? { ...s.run, state: data.state } : s.run,
          videoReused: false,
          downloadComplete: false,
          pendingReuse: false,
          navHistory: newNavHistory,
          navIndex: newNavHistory.length - 1,
        };
      });
      // Re-fetch full run for fresh fields (video_path, duration_sec after reuse copy)
      fetch(`${goUrl}/api/pipeline/runs/${id}`)
        .then((r) => r.json())
        .then((rdata: unknown) => {
          const { run: freshRun } = rdata as { run: Run };
          set((s) => ({
            run: s.run?.id === freshRun.id ? { ...freshRun, state: s.run!.state } : s.run,
          }));
        })
        .catch(() => { /* best-effort refresh */ });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'advance failed';
      log.error('[pipelineStore] advance failed', { error: msg });
      set({ error: msg });
    }
  },

  submitReviewHighlights: async (goUrl, id, req) => {
    log.info('[pipelineStore] submitReviewHighlights', { id });
    const payload: GatePayload = { kind: 'highlights', highlights: req.highlights };
    get().recordGateSubmission('WAITING_REVIEW_HIGHLIGHTS', payload);
    console.log('[pipelineStore] recordGateSubmission', 'WAITING_REVIEW_HIGHLIGHTS', payload);
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
    const payload: GatePayload = { kind: 'thumbnail', default_style: req.default_style, clip_overrides: req.clip_overrides };
    get().recordGateSubmission('WAITING_THUMBNAIL_CONFIG', payload);
    console.log('[pipelineStore] recordGateSubmission', 'WAITING_THUMBNAIL_CONFIG', payload);
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
    const payload: GatePayload = { kind: 'metadata', clips: req.clips };
    get().recordGateSubmission('WAITING_REVIEW_METADATA', payload);
    console.log('[pipelineStore] recordGateSubmission', 'WAITING_REVIEW_METADATA', payload);
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
    const payload: GatePayload = { kind: 'clips', selected_ids: req.selected_ids };
    get().recordGateSubmission('WAITING_REVIEW_CLIPS', payload);
    console.log('[pipelineStore] recordGateSubmission', 'WAITING_REVIEW_CLIPS', payload);
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
    const payload: GatePayload = { kind: 'upload', privacy: req.privacy };
    get().recordGateSubmission('WAITING_UPLOAD_CONFIRM', payload);
    console.log('[pipelineStore] recordGateSubmission', 'WAITING_UPLOAD_CONFIRM', payload);
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

  // ── Navigation ─────────────────────────────────────────────────────────────

  navigateBack: () => {
    const { navHistory, navIndex } = get();
    if (navIndex <= 0) return;
    let newIndex = navIndex - 1;
    // Skip compute states — user always lands on a gate step
    while (newIndex > 0 && !GATE_STATE_ORDER.includes(navHistory[newIndex])) {
      newIndex--;
    }
    set({ navIndex: newIndex });
    console.log('[pipelineStore] navigateBack', navHistory[newIndex]);
  },

  navigateForward: () => {
    const { navHistory, navIndex } = get();
    const newIndex = Math.min(navIndex + 1, navHistory.length - 1);
    set({ navIndex: newIndex });
    console.log('[pipelineStore] navigateForward', navHistory[newIndex]);
  },

  recordGateSubmission: (state, payload) => {
    set((s) => ({ gateHistory: { ...s.gateHistory, [state]: payload } }));
  },

  resubmitFromBacked: async (goUrl, targetGate, newPayload) => {
    const { activeRunId, gateHistory, advance, submitReviewHighlights, submitThumbnailConfig, submitReviewMetadata, submitReviewClips, submitUploadConfirm, cancelRun, clearRun, createRun, subscribeSSE } = get();

    log.info('[pipelineStore] resubmitFromBacked', { targetGate });
    console.log('[pipelineStore] resubmitFromBacked', targetGate, newPayload);

    // Snapshot gateHistory before clearRun wipes it
    const savedGateHistory = { ...gateHistory };

    // 1. Cancel current run if active
    if (activeRunId !== null) {
      await cancelRun(goUrl, activeRunId);
    }

    // 2. Clear state (resets navHistory, navIndex, gateHistory)
    clearRun();

    // 3. Create new run
    const newRunId = await createRun(goUrl);
    if (newRunId === null) {
      log.error('[pipelineStore] resubmitFromBacked: createRun failed');
      return;
    }
    subscribeSSE(goUrl, newRunId);

    // Helper: wait for a specific state via SSE (polling get().run?.state)
    const waitForState = (target: RunState, timeoutMs = 30000): Promise<boolean> =>
      new Promise((resolve) => {
        const deadline = Date.now() + timeoutMs;
        const check = () => {
          const current = get().run?.state;
          if (current === target) { resolve(true); return; }
          if (Date.now() > deadline) { resolve(false); return; }
          setTimeout(check, 100);
        };
        check();
      });

    // 4. Auto-replay all gates before targetGate
    const targetIdx = GATE_STATE_ORDER.indexOf(targetGate);
    for (let i = 0; i < targetIdx; i++) {
      const gate = GATE_STATE_ORDER[i];
      const hist = savedGateHistory[gate];
      if (!hist) continue;

      // Wait until we're at this gate
      const reached = await waitForState(gate);
      if (!reached) {
        log.error('[pipelineStore] resubmitFromBacked: timed out waiting for', { gate });
        return;
      }

      // Auto-advance through this gate
      if (hist.kind === 'url') {
        await advance(goUrl, newRunId, { url: hist.url });
      } else if (hist.kind === 'mode') {
        await advance(goUrl, newRunId, { url: hist.url, channel_id: hist.channel_id, mode_config: hist.mode_config });
      } else if (hist.kind === 'highlights') {
        await submitReviewHighlights(goUrl, newRunId, { highlights: hist.highlights });
      } else if (hist.kind === 'thumbnail') {
        await submitThumbnailConfig(goUrl, newRunId, { default_style: hist.default_style, clip_overrides: hist.clip_overrides });
      } else if (hist.kind === 'metadata') {
        await submitReviewMetadata(goUrl, newRunId, { clips: hist.clips });
      } else if (hist.kind === 'clips') {
        await submitReviewClips(goUrl, newRunId, { selected_ids: hist.selected_ids });
      } else if (hist.kind === 'upload') {
        await submitUploadConfirm(goUrl, newRunId, { privacy: hist.privacy });
      }
    }

    // 5. Wait for targetGate, then store newPayload so the step component pre-fills
    const reached = await waitForState(targetGate);
    if (!reached) {
      log.error('[pipelineStore] resubmitFromBacked: timed out waiting for targetGate', { targetGate });
      return;
    }
    get().recordGateSubmission(targetGate, newPayload);
    console.log('[pipelineStore] resubmitFromBacked complete — presenting targetGate', targetGate);
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

  // ── Preview ──────────────────────────────────────────────────────────────

  generatePreview: async (goUrl, runId, modeConfig) => {
    log.info('[pipelineStore] generating preview', { runId });
    set({ previewStatus: 'generating', previewPercent: 0, previewUrl: null, previewError: null });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${runId}/preview`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode_config: modeConfig }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Failed: ${res.status}`);
      }
      // SSE events will update previewStatus from here
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Preview generation failed';
      log.error('[pipelineStore] generatePreview failed', { error: msg });
      set({ previewStatus: 'error', previewError: msg, previewPercent: 0 });
    }
  },

  redownload: async (goUrl, runId) => {
    log.info('[pipelineStore] redownload', { runId });
    set({ downloadComplete: false, videoReused: false, previewStatus: 'idle', previewUrl: null });
    try {
      const res = await fetch(`${goUrl}/api/pipeline/runs/${runId}/redownload`, { method: 'POST' });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Redownload failed';
      log.error('[pipelineStore] redownload failed', { error: msg });
      set({ error: msg });
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
          const { pendingReuse } = get();

          // If pending reuse, skip auto-navigation to let StepUrl show reuse card
          if (pendingReuse && payload.state === 'WAITING_MODE') {
            set((s) => ({
              run: s.run?.id === payload.run_id ? { ...s.run, state: payload.state } : s.run,
            }));
          } else {
            set((s) => {
              // Dedup: advance() may have already pushed this state via HTTP response
              if (s.navHistory[s.navHistory.length - 1] === payload.state) {
                return {
                  run: s.run?.id === payload.run_id ? { ...s.run, state: payload.state } : s.run,
                };
              }
              const newNavHistory = [...s.navHistory, payload.state];
              return {
                run: s.run?.id === payload.run_id ? { ...s.run, state: payload.state } : s.run,
                navHistory: newNavHistory,
                navIndex: newNavHistory.length - 1,
              };
            });
          }
          // Re-fetch full run so fields like video_path/duration_sec are fresh
          fetch(`${goUrl}/api/pipeline/runs/${payload.run_id}`)
            .then((r) => r.json())
            .then((data: { run: Run }) => {
              set((s) => ({
                run: s.run?.id === data.run.id ? { ...data.run, state: s.run.state } : s.run,
              }));
            })
            .catch(() => { /* best-effort refresh */ });
        }

        if (evt.type === 'video_info') {
          const payload = evt.data as SSEVideoInfoPayload;
          set({
            videoInfo: {
              title: payload.title,
              thumbnailUrl: payload.thumbnail_url,
              durationSec: payload.duration_sec,
              channelName: payload.channel_name,
            },
          });
        }

        if (evt.type === 'download_complete') {
          const payload = evt.data as SSEDownloadCompletePayload;
          set((s) => ({
            downloadComplete: true,
            phaseProgress: null, // clear download progress
            run: s.run?.id === payload.run_id
              ? { ...s.run, video_path: payload.video_path }
              : s.run,
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

        if (evt.type === 'preview_progress') {
          const payload = evt.data as SSEPreviewProgressPayload;
          set({ previewStatus: 'generating', previewPercent: payload.percent });
        }

        if (evt.type === 'preview_ready') {
          const payload = evt.data as SSEPreviewReadyPayload;
          set({ previewStatus: 'ready', previewUrl: `${payload.url}?t=${Date.now()}`, previewPercent: 100 });
        }

        if (evt.type === 'preview_error') {
          const payload = evt.data as SSEPreviewErrorPayload;
          set({ previewStatus: 'error', previewError: payload.message, previewPercent: 0 });
        }

        if (evt.type === 'done') {
          es.close();
          const payload = evt.data as { run_id: number; state: RunState };
          set((s) => {
            const newNavHistory = [...s.navHistory, 'DONE' as RunState];
            return {
              run: s.run?.id === payload.run_id ? { ...s.run, state: 'DONE' } : s.run,
              sseCleanup: null,
              phaseProgress: null,
              navHistory: newNavHistory,
              navIndex: newNavHistory.length - 1,
            };
          });
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
