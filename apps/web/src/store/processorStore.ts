import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('processorStore');

export type JobStatus = 'idle' | 'running' | 'done' | 'error';

export interface ProcessorJob {
  status: JobStatus;
  logs: string[];
  result?: Record<string, unknown>;
}

type SseEvent = { type: 'log' | 'done' | 'error'; data: { message?: string; success?: boolean } };

interface ProcessorState {
  cutJobs: Record<string, ProcessorJob>;
  shortsJobs: Record<string, ProcessorJob>;
  optimizeJobs: Record<string, ProcessorJob>;
  transcriptJobs: Record<string, ProcessorJob>;
  analyzeJobs: Record<string, ProcessorJob>;
  startCut: (
    goUrl: string,
    payload: { input: string; start_sec: number; end_sec: number; output: string; session_id?: string },
  ) => Promise<string>;
  startShorts: (
    goUrl: string,
    payload: { input: string; output: string; width: number; height: number; session_id?: string },
  ) => Promise<string>;
  startOptimize: (
    goUrl: string,
    payload: { input: string; output: string; silence_threshold: number; session_id?: string },
  ) => Promise<string>;
  startTranscript: (
    goUrl: string,
    payload: { video_path: string; language: string; session_id?: string },
  ) => Promise<string>;
  startAnalyze: (
    goUrl: string,
    payload: { transcript_json: string; session_id?: string },
  ) => Promise<string>;
}

function makeSseAction<TPayload>(
  endpoint: string,
  jobsKey: keyof Pick<ProcessorState, 'cutJobs' | 'shortsJobs' | 'optimizeJobs' | 'transcriptJobs' | 'analyzeJobs'>,
  set: (fn: (state: ProcessorState) => Partial<ProcessorState>) => void,
): (goUrl: string, payload: TPayload) => Promise<string> {
  return async (goUrl: string, payload: TPayload): Promise<string> => {
    log.info(`starting ${endpoint}`, { payload });

    const res = await fetch(`${goUrl}/api/${endpoint}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    const { job_id } = (await res.json()) as { job_id: string };

    log.info(`${endpoint} job created`, { job_id });

    set((state) => ({
      [jobsKey]: { ...(state[jobsKey] as Record<string, ProcessorJob>), [job_id]: { status: 'running', logs: [] } },
    }));

    const es = new EventSource(`${goUrl}/api/${endpoint}/${job_id}/stream`);

    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data) as SseEvent;

        if (event.type === 'log') {
          set((state) => {
            const jobs = state[jobsKey] as Record<string, ProcessorJob>;
            const job = jobs[job_id];
            if (!job) return state;
            return {
              [jobsKey]: { ...jobs, [job_id]: { ...job, logs: [...job.logs, event.data.message ?? ''] } },
            };
          });
        } else if (event.type === 'done') {
          es.close();
          log.info(`${endpoint} completed`, { job_id });
          set((state) => {
            const jobs = state[jobsKey] as Record<string, ProcessorJob>;
            const job = jobs[job_id];
            if (!job) return state;
            return {
              [jobsKey]: { ...jobs, [job_id]: { ...job, status: 'done', result: event.data as Record<string, unknown> } },
            };
          });
        } else if (event.type === 'error') {
          es.close();
          log.error(`${endpoint} failed`, { job_id, message: event.data.message });
          set((state) => {
            const jobs = state[jobsKey] as Record<string, ProcessorJob>;
            const job = jobs[job_id];
            if (!job) return state;
            return {
              [jobsKey]: {
                ...jobs,
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
        log.error(`failed to parse SSE event for ${endpoint}`, { job_id });
      }
    };

    es.onerror = () => {
      es.close();
      log.error(`SSE connection error for ${endpoint}`, { job_id });
      set((state) => {
        const jobs = state[jobsKey] as Record<string, ProcessorJob>;
        const job = jobs[job_id];
        if (!job) return state;
        return { [jobsKey]: { ...jobs, [job_id]: { ...job, status: 'error' } } };
      });
    };

    return job_id;
  };
}

export const useProcessorStore = create<ProcessorState>((set) => ({
  cutJobs: {},
  shortsJobs: {},
  optimizeJobs: {},
  transcriptJobs: {},
  analyzeJobs: {},

  startCut: makeSseAction<{ input: string; start_sec: number; end_sec: number; output: string; session_id?: string }>(
    'cut',
    'cutJobs',
    set as (fn: (state: ProcessorState) => Partial<ProcessorState>) => void,
  ),

  startShorts: makeSseAction<{ input: string; output: string; width: number; height: number; session_id?: string }>(
    'shorts',
    'shortsJobs',
    set as (fn: (state: ProcessorState) => Partial<ProcessorState>) => void,
  ),

  startOptimize: makeSseAction<{ input: string; output: string; silence_threshold: number; session_id?: string }>(
    'optimize',
    'optimizeJobs',
    set as (fn: (state: ProcessorState) => Partial<ProcessorState>) => void,
  ),

  startTranscript: makeSseAction<{ video_path: string; language: string; session_id?: string }>(
    'transcript',
    'transcriptJobs',
    set as (fn: (state: ProcessorState) => Partial<ProcessorState>) => void,
  ),

  startAnalyze: makeSseAction<{ transcript_json: string; session_id?: string }>(
    'analyze',
    'analyzeJobs',
    set as (fn: (state: ProcessorState) => Partial<ProcessorState>) => void,
  ),
}));
