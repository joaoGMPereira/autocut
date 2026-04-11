import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('downloadStore');

export type JobStatus = 'idle' | 'running' | 'done' | 'error';

export interface DownloadJob {
  status: JobStatus;
  logs: string[];
  result?: Record<string, unknown>;
}

interface DownloadPayload {
  url: string;
  type: 'youtube' | 'twitch';
  output_dir: string;
  quality?: string;
  with_thumbnail?: boolean;
  with_metadata?: boolean;
  download_chat?: boolean;
  session_id?: string;
}

interface DownloadState {
  jobs: Record<string, DownloadJob>;
  startDownload: (goUrl: string, payload: DownloadPayload) => Promise<string>;
}

type SseEvent =
  | { type: 'log' | 'done' | 'error'; data: { message?: string; success?: boolean } }
  | { type: 'chat_downloaded'; data: { chat_path: string } };

export const useDownloadStore = create<DownloadState>((set) => ({
  jobs: {},

  startDownload: async (goUrl: string, payload: DownloadPayload): Promise<string> => {
    log.info('starting download', { url: payload.url, type: payload.type });

    const res = await fetch(`${goUrl}/api/download`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    const { job_id } = (await res.json()) as { job_id: string };

    log.info('download job created', { job_id });

    set((state) => ({
      jobs: { ...state.jobs, [job_id]: { status: 'running', logs: [] } },
    }));

    const es = new EventSource(`${goUrl}/api/download/${job_id}/stream`);

    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data) as SseEvent;

        if (event.type === 'log') {
          set((state) => {
            const job = state.jobs[job_id];
            if (!job) return state;
            return {
              jobs: { ...state.jobs, [job_id]: { ...job, logs: [...job.logs, event.data.message ?? ''] } },
            };
          });
        } else if (event.type === 'chat_downloaded') {
          log.info('chat downloaded', { job_id, chat_path: event.data.chat_path });
          set((state) => {
            const job = state.jobs[job_id];
            if (!job) return state;
            return {
              jobs: {
                ...state.jobs,
                [job_id]: { ...job, result: { ...job.result, chat_path: event.data.chat_path } },
              },
            };
          });
        } else if (event.type === 'done') {
          es.close();
          log.info('download completed', { job_id });
          set((state) => {
            const job = state.jobs[job_id];
            if (!job) return state;
            return {
              jobs: { ...state.jobs, [job_id]: { ...job, status: 'done', result: { ...job.result, ...(event.data as Record<string, unknown>) } } },
            };
          });
        } else if (event.type === 'error') {
          es.close();
          log.error('download failed', { job_id, message: event.data.message });
          set((state) => {
            const job = state.jobs[job_id];
            if (!job) return state;
            return {
              jobs: {
                ...state.jobs,
                [job_id]: {
                  ...job,
                  status: 'error',
                  logs: [...job.logs, `Error: ${event.data.message ?? 'Unknown'}`],
                },
              },
            };
          });
        }
      } catch {
        log.error('failed to parse SSE event', { job_id });
      }
    };

    es.onerror = () => {
      es.close();
      log.error('SSE connection error', { job_id });
      set((state) => {
        const job = state.jobs[job_id];
        if (!job) return state;
        return { jobs: { ...state.jobs, [job_id]: { ...job, status: 'error' } } };
      });
    };

    return job_id;
  },
}));
