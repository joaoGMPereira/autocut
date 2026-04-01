'use client';

import { useState } from 'react';
import { StepCard, type StepStatus } from '@/components/pipeline/StepCard';
import { LogViewer } from '@/components/pipeline/LogViewer';
import { useAppStore } from '@/store/appStore';
import { useDownloadStore } from '@/store/downloadStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { useDispatcherStore } from '@/store/dispatcherStore';

interface DownloadStepProps {
  status: StepStatus;
}

export function DownloadStep({ status }: DownloadStepProps) {
  const { goUrl } = useAppStore();
  const { jobs, startDownload } = useDownloadStore();
  const { run, activeRunId } = usePipelineStore();

  const prefillUrl = run?.url ?? '';
  const [url, setUrl] = useState(prefillUrl);
  const [platform, setPlatform] = useState<'youtube' | 'twitch'>('youtube');
  const [quality, setQuality] = useState('best');
  const [withThumbnail, setWithThumbnail] = useState(false);
  const [withMetadata, setWithMetadata] = useState(false);
  const [jobId, setJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const downloadTask = useDispatcherStore((s) =>
    activeRunId != null ? s.tasks[`download:${activeRunId}`] : undefined,
  );

  const job = jobId ? jobs[jobId] : null;
  // Prefer dispatcher task (background download) over local job state
  const liveStatus: StepStatus =
    downloadTask?.status === 'running' ? 'running' :
    downloadTask?.status === 'done'    ? 'done'    :
    downloadTask?.status === 'error'   ? 'error'   :
    job?.status === 'running' ? 'running' :
    job?.status === 'done'    ? 'done'    :
    job?.status === 'error'   ? 'error'   :
    status;
  const displayLogs = downloadTask?.logs ?? job?.logs ?? [];
  const isPreloaded = downloadTask != null &&
    (downloadTask.status === 'running' || downloadTask.status === 'done');

  const handleRun = async () => {
    setError(null);
    try {
      const id = await startDownload(goUrl, {
        url,
        type: platform,
        output_dir: '',
        quality,
        with_thumbnail: withThumbnail,
        with_metadata: withMetadata,
        ...(activeRunId != null ? { session_id: String(activeRunId) } : {}),
      });
      setJobId(id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Download failed');
    }
  };

  return (
    <StepCard
      title="Download"
      stepNumber={1}
      status={liveStatus}
      onRun={isPreloaded ? undefined : handleRun}
      runDisabled={!url}
      runLabel="Download"
    >
      {isPreloaded ? (
        <p className="text-xs text-[#5C5C80]">
          {downloadTask?.status === 'running'
            ? 'Baixando em background...'
            : 'Download concluído em background.'}
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          <label className="flex flex-col gap-1.5">
            <span className="text-xs text-[#5C5C80]">Video URL</span>
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://www.youtube.com/watch?v=..."
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 placeholder:text-[#5C5C80]"
            />
          </label>

          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1.5">
              <span className="text-xs text-[#5C5C80]">Platform</span>
              <select
                value={platform}
                onChange={(e) => setPlatform(e.target.value as 'youtube' | 'twitch')}
                className="rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
              >
                <option value="youtube">YouTube</option>
                <option value="twitch">Twitch</option>
              </select>
            </label>

            <label className="flex flex-col gap-1.5">
              <span className="text-xs text-[#5C5C80]">Quality</span>
              <select
                value={quality}
                onChange={(e) => setQuality(e.target.value)}
                className="rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
              >
                <option value="best">Best</option>
                <option value="1080p">1080p</option>
                <option value="720p">720p</option>
              </select>
            </label>
          </div>

          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-xs text-[#A0A0B8]">
              <input
                type="checkbox"
                checked={withThumbnail}
                onChange={(e) => setWithThumbnail(e.target.checked)}
                className="rounded border-border"
              />
              Download thumbnail
            </label>
            <label className="flex items-center gap-2 text-xs text-[#A0A0B8]">
              <input
                type="checkbox"
                checked={withMetadata}
                onChange={(e) => setWithMetadata(e.target.checked)}
                className="rounded border-border"
              />
              Save metadata
            </label>
          </div>
        </div>
      )}

      {error && <p className="text-xs text-destructive">{error}</p>}
      {displayLogs.length > 0 && <LogViewer logs={displayLogs} />}
    </StepCard>
  );
}
