'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { SubStepProgress } from '../SubStepProgress';
import type { ExecutePhase } from '@/types/pipeline';

const EXECUTING_PHASES: Array<{ phase: ExecutePhase; label: string; icon: string }> = [
  { phase: 'download', label: 'Downloading', icon: '⬇' },
  { phase: 'transcript', label: 'Transcribing', icon: '🎤' },
  { phase: 'analyze', label: 'Analyzing', icon: '🧠' },
  { phase: 'segment', label: 'Segmenting', icon: '✂' },
];

const GENERATING_PHASES: Array<{ phase: ExecutePhase; label: string; icon: string }> = [
  { phase: 'cut', label: 'Cutting clips', icon: '✂' },
  { phase: 'shorts', label: 'Generating shorts', icon: '📱' },
  { phase: 'thumbnail', label: 'Creating thumbnails', icon: '🖼' },
];

export function StepExecute() {
  const run = usePipelineStore((s) => s.run);
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);

  const isGenerating = run?.state === 'GENERATING_CLIPS';
  const phases = isGenerating ? GENERATING_PHASES : EXECUTING_PHASES;

  const title = isGenerating ? 'Generating Clips' : 'Processing';
  const subtitle = isGenerating
    ? 'Cutting clips, generating shorts, and creating thumbnails…'
    : 'Downloading, transcribing, and analyzing the video…';

  return (
    <div className="space-y-6 max-w-md">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">{title}</h2>
        <p className="text-sm text-zinc-400">{subtitle}</p>
      </div>

      <div className="space-y-1">
        {phases.map(({ phase, label, icon }) => {
          const isActive = phaseProgress?.phase === phase;
          const phaseDone = run?.active_phase
            ? EXECUTING_PHASES.concat(GENERATING_PHASES)
                .findIndex((p) => p.phase === run.active_phase) >
              EXECUTING_PHASES.concat(GENERATING_PHASES)
                .findIndex((p) => p.phase === phase)
            : false;

          return (
            <SubStepProgress
              key={phase}
              phase={phase}
              label={label}
              icon={icon}
              percentDone={isActive ? phaseProgress.percentDone : phaseDone ? 100 : 0}
              speedKbs={isActive ? phaseProgress.speedKbs : undefined}
              etaSec={isActive ? phaseProgress.etaSec : undefined}
              isActive={isActive}
            />
          );
        })}
      </div>

      <p className="text-xs text-zinc-500 italic">
        This may take several minutes. Results will appear automatically when ready.
      </p>
    </div>
  );
}
