'use client';

import { useState } from 'react';
import { StepCard, type StepStatus } from '@/components/pipeline/StepCard';
import { LogViewer } from '@/components/pipeline/LogViewer';
import { useAppStore } from '@/store/appStore';
import { useProcessorStore } from '@/store/processorStore';
import { usePipelineStore } from '@/store/pipelineStore';

interface OptimizeStepProps {
  status: StepStatus;
}

export function OptimizeStep({ status }: OptimizeStepProps) {
  const { goUrl } = useAppStore();
  const { optimizeJobs, startOptimize } = useProcessorStore();
  const { run, activeRunId } = usePipelineStore();

  const downloadOutput = run?.stepOutputs?.download as Record<string, unknown> | undefined;
  const prefillInput = (downloadOutput?.file_path as string) ?? '';

  const [input, setInput] = useState(prefillInput);
  const [strategy, setStrategy] = useState('complete');
  const [silenceThreshold, setSilenceThreshold] = useState(0.5);
  const [jobId, setJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const job = jobId ? optimizeJobs[jobId] : null;
  const liveStatus: StepStatus = job?.status === 'running' ? 'running' : job?.status === 'done' ? 'done' : job?.status === 'error' ? 'error' : status;

  const handleRun = async () => {
    setError(null);
    try {
      const id = await startOptimize(goUrl, {
        input: input || prefillInput,
        output: '',
        silence_threshold: silenceThreshold,
        ...(activeRunId != null ? { session_id: String(activeRunId) } : {}),
      });
      setJobId(id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Optimize failed');
    }
  };

  return (
    <StepCard title="Optimize" stepNumber={2} status={liveStatus} onRun={handleRun} runDisabled={!input && !prefillInput} runLabel="Optimize">
      <div className="flex flex-col gap-3">
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-[#5C5C80]">Input file</span>
          <input
            type="text"
            value={input || prefillInput}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Auto-filled from download step"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 placeholder:text-[#5C5C80]"
          />
        </label>

        <div className="grid grid-cols-2 gap-3">
          <label className="flex flex-col gap-1.5">
            <span className="text-xs text-[#5C5C80]">Strategy</span>
            <select
              value={strategy}
              onChange={(e) => setStrategy(e.target.value)}
              className="rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            >
              <option value="audio">Audio</option>
              <option value="transcript">Transcript</option>
              <option value="complete">Complete</option>
            </select>
          </label>

          <label className="flex flex-col gap-1.5">
            <span className="text-xs text-[#5C5C80]">Silence threshold</span>
            <input
              type="number"
              step={0.1}
              min={0}
              value={silenceThreshold}
              onChange={(e) => setSilenceThreshold(Number(e.target.value))}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </label>
        </div>
      </div>

      {error && <p className="text-xs text-destructive">{error}</p>}
      {job && <LogViewer logs={job.logs} />}
    </StepCard>
  );
}
