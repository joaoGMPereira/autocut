import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('effectsStore');

const GO_URL = process.env.NEXT_PUBLIC_GO_URL ?? 'http://127.0.0.1:4071';

export type JobStatus = 'idle' | 'running' | 'done' | 'error';

export interface EffectsJob {
  status: JobStatus;
  logs: string[];
}

export type EffectsJobKey =
  | 'antiDupJob'
  | 'speedJob'
  | 'transitionJob'
  | 'textJob'
  | 'overlayJob';

type SseEvent = {
  type: 'progress' | 'done' | 'error' | 'ping';
  data?: { message?: string };
};

interface EffectsState {
  antiDupJob: EffectsJob;
  speedJob: EffectsJob;
  transitionJob: EffectsJob;
  textJob: EffectsJob;
  overlayJob: EffectsJob;
  applyEffect: (endpoint: string, body: object, jobKey: EffectsJobKey) => Promise<void>;
}

const IDLE_JOB: EffectsJob = { status: 'idle', logs: [] };

export const useEffectsStore = create<EffectsState>((set) => ({
  antiDupJob: { ...IDLE_JOB },
  speedJob: { ...IDLE_JOB },
  transitionJob: { ...IDLE_JOB },
  textJob: { ...IDLE_JOB },
  overlayJob: { ...IDLE_JOB },

  applyEffect: async (endpoint, body, jobKey) => {
    log.info(`applying effect`, { endpoint, jobKey });

    set({ [jobKey]: { status: 'running', logs: [] } });

    let res: Response;
    try {
      res = await fetch(`${GO_URL}/${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
    } catch (err) {
      log.error('fetch failed', { endpoint, err });
      set({ [jobKey]: { status: 'error', logs: ['Network error: failed to connect'] } });
      return;
    }

    if (!res.ok) {
      log.error('POST failed', { endpoint, status: res.status });
      set({ [jobKey]: { status: 'error', logs: [`Server error: ${res.status}`] } });
      return;
    }

    let job_id: string;
    try {
      const json = (await res.json()) as { job_id: string };
      job_id = json.job_id;
    } catch (err) {
      log.error('failed to parse job_id', { endpoint, err });
      set({ [jobKey]: { status: 'error', logs: ['Failed to parse server response'] } });
      return;
    }

    log.info('job created', { endpoint, job_id, jobKey });

    const streamUrl = `${GO_URL}/api/effects/${job_id}/stream`;
    const es = new EventSource(streamUrl);

    const appendLog = (msg: string) => {
      set((state) => {
        const job = state[jobKey];
        return { [jobKey]: { ...job, logs: [...job.logs, msg] } };
      });
    };

    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data as string) as SseEvent;

        if (event.type === 'progress') {
          appendLog(event.data?.message ?? '');
        } else if (event.type === 'done') {
          es.close();
          log.info('effect completed', { job_id, jobKey });
          set((state) => {
            const job = state[jobKey];
            return { [jobKey]: { ...job, status: 'done' } };
          });
        } else if (event.type === 'error') {
          es.close();
          const msg = event.data?.message ?? 'Unknown error';
          log.error('effect failed', { job_id, jobKey, msg });
          set((state) => {
            const job = state[jobKey];
            return { [jobKey]: { ...job, status: 'error', logs: [...job.logs, `Error: ${msg}`] } };
          });
        }
        // 'ping' — no-op keepalive
      } catch {
        log.error('failed to parse SSE event', { job_id, jobKey });
      }
    };

    es.onerror = () => {
      es.close();
      log.error('SSE connection error', { job_id, jobKey });
      set((state) => {
        const job = state[jobKey];
        return { [jobKey]: { ...job, status: 'error', logs: [...job.logs, 'SSE connection error'] } };
      });
    };
  },
}));
