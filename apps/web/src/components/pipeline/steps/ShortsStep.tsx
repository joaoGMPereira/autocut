'use client';

import { useState } from 'react';
import { StepCard, type StepStatus } from '@/components/pipeline/StepCard';
import { LogViewer } from '@/components/pipeline/LogViewer';
import { useAppStore } from '@/store/appStore';
import { useProcessorStore } from '@/store/processorStore';

interface ShortsStepProps {
  status: StepStatus;
}

export function ShortsStep({ status }: ShortsStepProps) {
  const { goUrl } = useAppStore();
  const { shortsJobs, startShorts } = useProcessorStore();

  const [count, setCount] = useState(3);
  const [minDuration, setMinDuration] = useState(15);
  const [maxDuration, setMaxDuration] = useState(60);
  const [jobId, setJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const job = jobId ? shortsJobs[jobId] : null;
  const liveStatus: StepStatus = job?.status === 'running' ? 'running' : job?.status === 'done' ? 'done' : job?.status === 'error' ? 'error' : status;

  const handleRun = async () => {
    setError(null);
    try {
      const id = await startShorts(goUrl, {
        input: '',
        output: '',
        width: 1080,
        height: 1920,
      });
      setJobId(id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Shorts generation failed');
    }
  };

  return (
    <StepCard title="Shorts" stepNumber={5} status={liveStatus} onRun={handleRun} runLabel="Generate Shorts">
      <div className="flex flex-col gap-3">
        <div className="grid grid-cols-3 gap-3">
          <label className="flex flex-col gap-1.5">
            <span className="text-xs text-[#5C5C80]">Count</span>
            <input
              type="number"
              min={1}
              value={count}
              onChange={(e) => setCount(Number(e.target.value))}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </label>
          <label className="flex flex-col gap-1.5">
            <span className="text-xs text-[#5C5C80]">Min duration (s)</span>
            <input
              type="number"
              min={1}
              value={minDuration}
              onChange={(e) => setMinDuration(Number(e.target.value))}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </label>
          <label className="flex flex-col gap-1.5">
            <span className="text-xs text-[#5C5C80]">Max duration (s)</span>
            <input
              type="number"
              min={1}
              value={maxDuration}
              onChange={(e) => setMaxDuration(Number(e.target.value))}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </label>
        </div>
        <p className="text-xs text-[#5C5C80]">Shorts generation from analyzed segments (placeholder).</p>
      </div>

      {error && <p className="text-xs text-destructive">{error}</p>}
      {job && <LogViewer logs={job.logs} />}
    </StepCard>
  );
}
