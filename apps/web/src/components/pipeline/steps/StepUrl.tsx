'use client';

import { useEffect, useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import type { PhaseProgress, VideoInfo } from '@/store/pipelineStore';
import { useUrlHistoryStore } from '@/store/urlHistoryStore';
import type { GatePayload, UrlHistoryEntry } from '@/types/pipeline';

function DownloadProgressCard({ videoInfo, phaseProgress }: { videoInfo: VideoInfo | null; phaseProgress: PhaseProgress | null }) {
  const pct = phaseProgress?.percentDone ?? 0;
  return (
    <div className="rounded-md border border-border bg-background p-4 space-y-3">
      {videoInfo && (
        <div className="flex gap-3 items-start">
          {videoInfo.thumbnailUrl && (
            <img
              src={videoInfo.thumbnailUrl}
              alt=""
              className="w-20 h-12 object-cover rounded shrink-0"
            />
          )}
          <div className="min-w-0">
            <p className="text-sm font-medium text-foreground truncate">{videoInfo.title}</p>
            <p className="text-xs text-zinc-500">{videoInfo.durationSec}s</p>
          </div>
        </div>
      )}
      <div className="space-y-1">
        <div className="flex justify-between text-xs text-zinc-500">
          <span>Downloading…</span>
          <span>{Math.round(pct)}%</span>
        </div>
        <div className="h-1.5 rounded-full bg-zinc-800 overflow-hidden">
          <div
            className="h-full rounded-full bg-zinc-300 transition-all duration-300"
            style={{ width: `${pct}%` }}
          />
        </div>
        {(phaseProgress?.speedKbs != null || phaseProgress?.etaSec != null) && (
          <div className="flex gap-3 text-xs text-zinc-600">
            {phaseProgress.speedKbs != null && <span>{phaseProgress.speedKbs} KB/s</span>}
            {phaseProgress.etaSec != null && <span>ETA {phaseProgress.etaSec}s</span>}
          </div>
        )}
      </div>
    </div>
  );
}

function PreparingCard() {
  return (
    <div className="rounded-md border border-border bg-background p-4">
      <div className="flex items-center gap-2 text-sm text-zinc-500">
        <span className="inline-block h-3 w-3 rounded-full border-2 border-zinc-500 border-t-transparent animate-spin" />
        <span>Preparing download…</span>
      </div>
    </div>
  );
}

function UrlHistorySection({ goUrl, onSelect }: { goUrl: string; onSelect: (url: string) => void }) {
  const history = useUrlHistoryStore((s) => s.history);
  const removeEntry = useUrlHistoryStore((s) => s.removeEntry);
  const clearAll = useUrlHistoryStore((s) => s.clearAll);

  if (history.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-zinc-500">Recent URLs</span>
        <button
          onClick={() => void clearAll(goUrl)}
          className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
        >
          Clear all
        </button>
      </div>
      <ul className="space-y-1">
        {history.map((entry: UrlHistoryEntry) => (
          <li key={entry.id} className="flex items-center gap-2 group">
            <button
              onClick={() => onSelect(entry.url)}
              className="flex-1 min-w-0 text-left rounded px-2 py-1.5 hover:bg-zinc-800 transition-colors"
            >
              {entry.video_title && (
                <p className="text-xs font-medium text-foreground truncate">{entry.video_title}</p>
              )}
              <p className="text-xs text-zinc-500 truncate">{entry.url}</p>
            </button>
            <button
              onClick={() => void removeEntry(goUrl, entry.id)}
              className="shrink-0 text-zinc-700 hover:text-zinc-400 transition-colors opacity-0 group-hover:opacity-100 px-1"
              aria-label="Remove"
            >
              ×
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

interface StepUrlProps {
  historical?: GatePayload;
}

export function StepUrl({ historical }: StepUrlProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const createRun = usePipelineStore((s) => s.createRun);
  const advance = usePipelineStore((s) => s.advance);
  const subscribeSSE = usePipelineStore((s) => s.subscribeSSE);
  const resubmitFromBacked = usePipelineStore((s) => s.resubmitFromBacked);
  const isLoading = usePipelineStore((s) => s.isLoading);
  const error = usePipelineStore((s) => s.error);
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);
  const videoInfo = usePipelineStore((s) => s.videoInfo);
  const fetchHistory = useUrlHistoryStore((s) => s.fetchHistory);

  const historicalUrl = historical?.kind === 'url' ? historical.url : undefined;
  const [url, setUrl] = useState(historicalUrl ?? '');
  // True between "Start" click and the first phase_progress SSE event
  const [isPending, setIsPending] = useState(false);

  useEffect(() => {
    void fetchHistory(goUrl);
  }, [goUrl, fetchHistory]);

  // Clear pending once download actually starts (SSE fires)
  useEffect(() => {
    if (phaseProgress !== null) setIsPending(false);
  }, [phaseProgress]);

  const isHistorical = historical !== undefined;
  // Only show download UI when NOT navigated back — phaseProgress may still be
  // populated from the original run when the user clicks Back to this step.
  const isDownloading = !isHistorical && phaseProgress?.phase === 'download';
  const showPreparing = !isHistorical && isPending && !isDownloading;

  const handleStart = async () => {
    const trimmed = url.trim();
    if (!trimmed) return;
    setIsPending(true);
    const runId = await createRun(goUrl);
    if (!runId) { setIsPending(false); return; }
    subscribeSSE(goUrl, runId);
    await advance(goUrl, runId, { url: trimmed });
  };

  const handleResubmit = async () => {
    const trimmed = url.trim();
    if (!trimmed) return;
    await resubmitFromBacked(goUrl, 'WAITING_URL', { kind: 'url', url: trimmed });
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !isLoading && !isDownloading && !showPreparing) {
      if (isHistorical) void handleResubmit();
      else void handleStart();
    }
  };

  return (
    <div className="space-y-6 max-w-lg">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold text-foreground">Video URL</h2>
        <p className="text-sm text-zinc-500">Paste a YouTube or Twitch URL to start the pipeline.</p>
      </div>

      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}

      {isDownloading ? (
        <DownloadProgressCard videoInfo={videoInfo} phaseProgress={phaseProgress} />
      ) : showPreparing ? (
        <PreparingCard />
      ) : (
        <div className="space-y-3">
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="https://www.youtube.com/watch?v=..."
            disabled={isLoading}
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500 disabled:opacity-50"
          />

          <button
            onClick={() => void (isHistorical ? handleResubmit() : handleStart())}
            disabled={isLoading || !url.trim()}
            className="w-full rounded-md bg-zinc-100 px-4 py-2 text-sm font-medium text-zinc-900 transition-colors hover:bg-zinc-200 disabled:cursor-not-allowed disabled:opacity-40"
          >
            {isLoading ? 'Starting…' : isHistorical ? 'Re-submit' : 'Start Pipeline'}
          </button>

          {error && (
            <p className="text-xs text-red-400">{error}</p>
          )}

          {!isHistorical && <UrlHistorySection goUrl={goUrl} onSelect={(u) => setUrl(u)} />}
        </div>
      )}
    </div>
  );
}
