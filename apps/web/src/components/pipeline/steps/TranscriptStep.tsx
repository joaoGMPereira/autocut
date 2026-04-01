'use client';

import { useState } from 'react';
import { StepCard, type StepStatus } from '@/components/pipeline/StepCard';
import { LogViewer } from '@/components/pipeline/LogViewer';
import { useAppStore } from '@/store/appStore';
import { useProcessorStore } from '@/store/processorStore';
import { usePipelineStore } from '@/store/pipelineStore';

interface TranscriptStepProps {
  status: StepStatus;
}

export function TranscriptStep({ status }: TranscriptStepProps) {
  const { goUrl } = useAppStore();
  const { transcriptJobs, startTranscript } = useProcessorStore();
  const { run, activeRunId } = usePipelineStore();

  const optimizeOutput = run?.stepOutputs?.optimize as Record<string, unknown> | undefined;
  const downloadOutput = run?.stepOutputs?.download as Record<string, unknown> | undefined;
  const prefillPath = (optimizeOutput?.file_path as string) ?? (downloadOutput?.file_path as string) ?? '';

  const [videoPath, setVideoPath] = useState(prefillPath);
  const [language, setLanguage] = useState('auto');
  const [jobId, setJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const job = jobId ? transcriptJobs[jobId] : null;
  const liveStatus: StepStatus = job?.status === 'running' ? 'running' : job?.status === 'done' ? 'done' : job?.status === 'error' ? 'error' : status;

  const handleRun = async () => {
    setError(null);
    try {
      const id = await startTranscript(goUrl, {
        video_path: videoPath || prefillPath,
        language,
        ...(activeRunId != null ? { session_id: String(activeRunId) } : {}),
      });
      setJobId(id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Transcript failed');
    }
  };

  return (
    <StepCard title="Transcript" stepNumber={3} status={liveStatus} onRun={handleRun} runDisabled={!videoPath && !prefillPath} runLabel="Transcribe">
      <div className="flex flex-col gap-3">
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-[#5C5C80]">Video file</span>
          <input
            type="text"
            value={videoPath || prefillPath}
            onChange={(e) => setVideoPath(e.target.value)}
            placeholder="Auto-filled from previous step"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 placeholder:text-[#5C5C80]"
          />
        </label>

        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-[#5C5C80]">Language</span>
          <select
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            className="rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
          >
            <option value="auto">Auto-detect</option>
            <option value="pt">Portuguese (pt)</option>
            <option value="en">English (en)</option>
          </select>
        </label>
      </div>

      {error && <p className="text-xs text-destructive">{error}</p>}
      {job && <LogViewer logs={job.logs} />}
    </StepCard>
  );
}
