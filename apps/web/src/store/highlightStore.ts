import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('highlightStore');

export interface Highlight {
  startSec: number;
  endSec: number;
  confidence: number;
  strategies: string[];
  scores: Record<string, number>;
}

export type HighlightStatus = 'idle' | 'running' | 'done' | 'error';

interface HighlightState {
  highlights: Highlight[];
  strategies: string[];
  threshold: number;
  chatJsonPath: string;
  status: HighlightStatus;
  toggleStrategy: (s: string) => void;
  setThreshold: (v: number) => void;
  setChatJsonPath: (p: string) => void;
  runDetection: (videoPath: string, goUrl: string) => Promise<void>;
  reset: () => void;
}

// Normalize SSE highlight event — handles both legacy (start/end/score)
// and multi-strategy (start_sec/end_sec/confidence/strategies/scores) shapes.
function normalizeHighlight(raw: Record<string, unknown>): Highlight {
  const startSec =
    typeof raw['start_sec'] === 'number'
      ? raw['start_sec']
      : typeof raw['start'] === 'number'
        ? raw['start']
        : 0;
  const endSec =
    typeof raw['end_sec'] === 'number'
      ? raw['end_sec']
      : typeof raw['end'] === 'number'
        ? raw['end']
        : startSec + 5;
  const confidence =
    typeof raw['confidence'] === 'number'
      ? raw['confidence']
      : typeof raw['score'] === 'number'
        ? raw['score']
        : 0;
  const strategies = Array.isArray(raw['strategies'])
    ? (raw['strategies'] as string[])
    : ['ai'];
  const scores =
    raw['scores'] != null && typeof raw['scores'] === 'object'
      ? (raw['scores'] as Record<string, number>)
      : { ai: confidence };
  return { startSec, endSec, confidence, strategies, scores };
}

export const useHighlightStore = create<HighlightState>((set, get) => ({
  highlights: [],
  strategies: ['ai'],
  threshold: 0.5,
  chatJsonPath: '',
  status: 'idle',

  toggleStrategy: (s: string) => {
    const current = get().strategies;
    if (current.includes(s)) {
      set({ strategies: current.filter((x) => x !== s) });
    } else {
      set({ strategies: [...current, s] });
    }
  },

  setThreshold: (v: number) => set({ threshold: v }),

  setChatJsonPath: (p: string) => set({ chatJsonPath: p }),

  reset: () => set({ highlights: [], status: 'idle' }),

  runDetection: async (videoPath: string, goUrl: string) => {
    const { strategies, chatJsonPath } = get();

    if (strategies.length === 0) {
      log.warn('runDetection called with no strategies selected');
      return;
    }

    set({ highlights: [], status: 'running' });
    log.info('starting highlight detection', { videoPath, strategies });

    let jobId: string;
    try {
      const body: Record<string, unknown> = { video_path: videoPath, strategies };
      if (strategies.includes('chat') && chatJsonPath) {
        body['chat_json_path'] = chatJsonPath;
      }

      const res = await fetch(`${goUrl}/api/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const text = await res.text().catch(() => res.statusText);
        throw new Error(`POST /api/analyze failed: ${res.status} ${text}`);
      }

      const json = (await res.json()) as { job_id: string };
      jobId = json.job_id;
      log.info('detection job created', { jobId });
    } catch (err) {
      log.error('failed to start detection', { err });
      set({ status: 'error' });
      return;
    }

    const es = new EventSource(`${goUrl}/api/analyze/${jobId}/stream`);

    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data as string) as Record<string, unknown>;
        const type = event['type'];

        if (type === 'highlight') {
          const h = normalizeHighlight(event);
          log.debug('highlight received', { startSec: h.startSec, confidence: h.confidence });
          set((state) => ({ highlights: [...state.highlights, h] }));
        } else if (type === 'done') {
          es.close();
          log.info('detection complete', { total: get().highlights.length });
          set({ status: 'done' });
        } else if (type === 'error') {
          es.close();
          const msg = typeof event['message'] === 'string' ? event['message'] : 'Unknown error';
          log.error('detection error from server', { msg });
          set({ status: 'error' });
        }
      } catch {
        log.error('failed to parse SSE event', { jobId });
      }
    };

    es.onerror = () => {
      es.close();
      log.error('SSE connection error', { jobId });
      if (get().status === 'running') {
        set({ status: 'error' });
      }
    };
  },
}));
