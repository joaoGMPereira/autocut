'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import type { StepStatus } from '@/components/pipeline/StepCard';
import { DownloadStep } from './DownloadStep';
import { OptimizeStep } from './OptimizeStep';
import { TranscriptStep } from './TranscriptStep';
import { AnalyzeStep } from './AnalyzeStep';
import { StepConnector } from '@/components/pipeline/StepConnector';

function useStepStatus(stepName: string): StepStatus {
  const run = usePipelineStore((s) => s.run);
  if (!run) return 'idle';
  const step = run.steps.find((s) => s.step === stepName);
  if (!step) return 'idle';
  if (step.status === 'running') return 'running';
  if (step.status === 'done' || step.status === 'completed') return 'done';
  if (step.status === 'error' || step.status === 'failed') return 'error';
  if (step.status === 'skipped') return 'skipped';
  return 'idle';
}

function useConnectorStatus(predecessorStep: string): 'idle' | 'active' {
  const status = useStepStatus(predecessorStep);
  return status === 'done' ? 'active' : 'idle';
}

function useConnectorLabel(predecessorStep: string): string | undefined {
  const run = usePipelineStore((s) => s.run);
  const status = useStepStatus(predecessorStep);
  if (status !== 'done' || !run?.stepOutputs) return undefined;
  const output = run.stepOutputs[predecessorStep] as Record<string, unknown> | undefined;
  if (!output) return undefined;
  const filePath = (output.file_path as string) ?? (output.json_path as string);
  if (!filePath) return undefined;
  return filePath.split('/').pop();
}

export function ExecuteStepPanel() {
  const downloadStatus = useStepStatus('download');
  const optimizeStatus = useStepStatus('optimize');
  const transcriptStatus = useStepStatus('transcript');
  const analyzeStatus = useStepStatus('analyze');

  const connDownload = useConnectorStatus('download');
  const connOptimize = useConnectorStatus('optimize');
  const connTranscript = useConnectorStatus('transcript');

  const labelDownload = useConnectorLabel('download');
  const labelOptimize = useConnectorLabel('optimize');
  const labelTranscript = useConnectorLabel('transcript');

  return (
    <div className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold text-[#F0F0F8] mb-2">Execucao</h2>
      <div className="flex flex-col items-stretch gap-0">
        <DownloadStep status={downloadStatus} />
        <StepConnector status={connDownload} label={labelDownload} />
        <OptimizeStep status={optimizeStatus} />
        <StepConnector status={connOptimize} label={labelOptimize} />
        <TranscriptStep status={transcriptStatus} />
        <StepConnector status={connTranscript} label={labelTranscript} />
        <AnalyzeStep status={analyzeStatus} />
      </div>
    </div>
  );
}
