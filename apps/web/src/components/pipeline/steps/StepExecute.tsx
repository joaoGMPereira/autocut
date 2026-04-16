'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { SubStepProgress } from '@/components/pipeline/SubStepProgress';

export function StepExecute() {
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);

  // Transcript phase is done once we've moved on to the analyze phase.
  const transcriptDone =
    phaseProgress?.phase === 'analyze' ||
    (phaseProgress == null ? false : !['transcript'].includes(phaseProgress.phase));

  const transcriptPct =
    phaseProgress?.phase === 'transcript'
      ? phaseProgress.percentDone
      : transcriptDone
      ? 100
      : 0;

  const analyzePct =
    phaseProgress?.phase === 'analyze' ? phaseProgress.percentDone : 0;

  console.log('[StepExecute] phaseProgress', phaseProgress);

  return (
    <div className="space-y-4 max-w-lg">
      <h2 className="text-xl font-semibold text-foreground">Processing</h2>
      <SubStepProgress
        phase="transcript"
        label="Transcription"
        percentDone={transcriptPct}
        isActive={phaseProgress?.phase === 'transcript'}
      />
      <SubStepProgress
        phase="analyze"
        label="Analysis"
        percentDone={analyzePct}
        isActive={phaseProgress?.phase === 'analyze'}
      />
    </div>
  );
}
