'use client';

import { useState } from 'react';
import { StepCard, type StepStatus } from '@/components/pipeline/StepCard';
import { LogViewer } from '@/components/pipeline/LogViewer';
import { useAppStore } from '@/store/appStore';
import { useProcessorStore } from '@/store/processorStore';
import { usePipelineStore } from '@/store/pipelineStore';

interface AnalyzeStepProps {
  status: StepStatus;
}

export function AnalyzeStep({ status }: AnalyzeStepProps) {
  const { goUrl } = useAppStore();
  const { analyzeJobs, startAnalyze } = useProcessorStore();
  const { run, activeRunId } = usePipelineStore();

  const transcriptOutput = run?.stepOutputs?.transcript as Record<string, unknown> | undefined;
  const prefillJson = (transcriptOutput?.json_path as string) ?? '';

  const [transcriptJson, setTranscriptJson] = useState(prefillJson);
  const [jobId, setJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const job = jobId ? analyzeJobs[jobId] : null;
  const liveStatus: StepStatus = job?.status === 'running' ? 'running' : job?.status === 'done' ? 'done' : job?.status === 'error' ? 'error' : status;

  const handleRun = async () => {
    setError(null);
    try {
      const id = await startAnalyze(goUrl, {
        transcript_json: transcriptJson || prefillJson,
        ...(activeRunId != null ? { session_id: String(activeRunId) } : {}),
      });
      setJobId(id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Analyze failed');
    }
  };

  return (
    <StepCard title="Analyze" stepNumber={4} status={liveStatus} onRun={handleRun} runDisabled={!transcriptJson && !prefillJson} runLabel="Analyze">
      <div className="flex flex-col gap-3">
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-[#5C5C80]">Transcript JSON</span>
          <input
            type="text"
            value={transcriptJson || prefillJson}
            onChange={(e) => setTranscriptJson(e.target.value)}
            placeholder="Auto-filled from transcript step"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 placeholder:text-[#5C5C80]"
          />
        </label>
      </div>

      {error && <p className="text-xs text-destructive">{error}</p>}
      {job && <LogViewer logs={job.logs} />}
    </StepCard>
  );
}
