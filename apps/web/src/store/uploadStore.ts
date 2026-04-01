import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('uploadStore');

export type JobStatus = 'idle' | 'running' | 'done' | 'error';

export interface Upload {
  ID: number;
  VideoPath: string;
  Status: string;
  CreatedAt: string;
}

export interface UploadJob {
  status: JobStatus;
  logs: string[];
}

interface UploadPayload {
  video_path: string;
  channel_id: number;
  title: string;
  description: string;
}

interface UploadState {
  uploads: Upload[];
  uploadJobs: Record<string, UploadJob>;
  activeJobId: string | null;
  fetchUploads: (goUrl: string) => Promise<void>;
  startUpload: (goUrl: string, payload: UploadPayload) => Promise<string>;
}

export const useUploadStore = create<UploadState>((set, get) => ({
  uploads: [],
  uploadJobs: {},
  activeJobId: null,

  fetchUploads: async (goUrl) => {
    try {
      const res = await fetch(`${goUrl}/api/upload`);
      if (!res.ok) return;
      const data = (await res.json()) as Upload[];
      set({ uploads: data });
    } catch (err) {
      log.error('fetch uploads failed', { err });
    }
  },

  startUpload: async (goUrl, payload) => {
    const res = await fetch(`${goUrl}/api/upload`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    const { job_id } = (await res.json()) as { job_id: string };

    log.info('upload started', { job_id });
    set((state) => ({
      uploadJobs: { ...state.uploadJobs, [job_id]: { status: 'running', logs: [] } },
      activeJobId: job_id,
    }));

    const es = new EventSource(`${goUrl}/api/upload/${job_id}/stream`);

    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data) as {
          type: 'log' | 'done' | 'error';
          data: { message?: string; success?: boolean };
        };

        if (event.type === 'log') {
          set((state) => {
            const job = state.uploadJobs[job_id];
            if (!job) return state;
            return {
              uploadJobs: {
                ...state.uploadJobs,
                [job_id]: { ...job, logs: [...job.logs, event.data.message ?? ''] },
              },
            };
          });
        } else if (event.type === 'done') {
          es.close();
          set((state) => {
            const job = state.uploadJobs[job_id];
            if (!job) return state;
            return {
              uploadJobs: {
                ...state.uploadJobs,
                [job_id]: { ...job, status: event.data.success ? 'done' : 'error' },
              },
            };
          });
          void get().fetchUploads(goUrl);
        } else if (event.type === 'error') {
          es.close();
          set((state) => {
            const job = state.uploadJobs[job_id];
            if (!job) return state;
            return {
              uploadJobs: {
                ...state.uploadJobs,
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
        /* ignore parse errors */
      }
    };

    es.onerror = () => {
      es.close();
      set((state) => {
        const job = state.uploadJobs[job_id];
        if (!job) return state;
        return {
          uploadJobs: { ...state.uploadJobs, [job_id]: { ...job, status: 'error' } },
        };
      });
    };

    return job_id;
  },
}));
