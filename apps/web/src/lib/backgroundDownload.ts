import { useDownloadStore } from '@/store/downloadStore';
import { useDispatcherStore } from '@/store/dispatcherStore';

/** Minimal run shape needed for hydration — avoids a circular import with pipelineStore. */
interface RunForHydration {
  id: number;
  stepOutputs?: Record<string, unknown>;
  steps?: Array<{ step: string; status: string; jobId: string }>;
}

/**
 * Pre-populate the dispatcher with the download status of an existing run.
 * Call after loading a run from the API so consumers see "done" without re-downloading.
 */
export function hydrateDispatcherFromRun(run: RunForHydration): void {
  const key = `download:${run.id}`;
  const dispatcher = useDispatcherStore.getState();

  // Don't overwrite an in-progress task from the current session
  const existing = dispatcher.tasks[key];
  if (existing?.status === 'running') return;

  const downloadOutput = run.stepOutputs?.download as { file_path?: string } | undefined;

  if (downloadOutput?.file_path) {
    // Download completed in a previous session
    dispatcher.markDone(key, downloadOutput as Record<string, unknown>);
    return;
  }

  // Fall back to checking run.steps
  const downloadStep = run.steps?.find((s) => s.step === 'download');
  if (downloadStep?.status === 'done') {
    dispatcher.markDone(key, {});
  }
  // Note: reconnecting to an in-progress job from a previous session is not
  // handled here — that case is rare and the user can refresh the page.
}

/** Parse yt-dlp progress percentage from accumulated logs. */
function parseYtDlpProgress(logs: string[]): number {
  for (let i = logs.length - 1; i >= 0; i--) {
    const match = logs[i].match(/\[download\]\s+([\d.]+)%/);
    if (match) return parseFloat(match[1]);
  }
  return 0;
}

/**
 * Start a background download for a pipeline run.
 * Task key: `download:${runId}` — consumers can read it from dispatcherStore.
 */
export async function startBackgroundDownload(
  goUrl: string,
  runId: number,
  url: string,
): Promise<void> {
  const key = `download:${runId}`;
  useDispatcherStore
    .getState()
    .upsertTask(key, { status: 'running', progress: 0, logs: [], data: {} });

  let jobId: string;
  try {
    jobId = await useDownloadStore.getState().startDownload(goUrl, {
      url,
      type: 'youtube',
      output_dir: '',
      quality: 'best',
      with_thumbnail: true,
      with_metadata: true,
      session_id: String(runId),
    });
  } catch (err) {
    useDispatcherStore
      .getState()
      .markError(key, err instanceof Error ? err.message : 'Download failed');
    return;
  }

  let finished = false;
  const unsub = useDownloadStore.subscribe((state) => {
    if (finished) return;
    const job = state.jobs[jobId];
    if (!job) return;

    const progress = parseYtDlpProgress(job.logs);
    useDispatcherStore.getState().upsertTask(key, { progress, logs: job.logs });

    if (job.status === 'done') {
      finished = true;
      useDispatcherStore.getState().markDone(key, job.result ?? {});
      unsub();
    } else if (job.status === 'error') {
      finished = true;
      useDispatcherStore
        .getState()
        .markError(key, job.logs.at(-1) ?? 'Download error');
      unsub();
    }
  });
}
