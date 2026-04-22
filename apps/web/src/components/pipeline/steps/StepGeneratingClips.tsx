'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { SubStepProgress } from '../SubStepProgress';

export function StepGeneratingClips() {
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);
  const clipProgress = usePipelineStore((s) => s.clipProgress);
  const entries = clipProgress ? Object.entries(clipProgress) : [];

  const overallPct = phaseProgress?.phase === 'cut' ? phaseProgress.percentDone : 0;
  const isEffectsPhase = overallPct >= 50 || entries.length > 0;

  return (
    <div className="space-y-4 max-w-xl">
      <div>
        <h2 className="text-xl font-semibold">Cutting Clips</h2>
        <p className="text-sm text-zinc-500 mt-1">
          {isEffectsPhase ? 'Applying effects to each clip…' : 'Cutting video into segments…'}
        </p>
      </div>

      <SubStepProgress
        phase="cut"
        label={isEffectsPhase ? 'Applying effects' : 'Splitting video'}
        percentDone={overallPct}
        isActive={true}
      />

      {entries.length > 0 && (
        <div className="space-y-2 pl-2">
          {entries.map(([clipId, pct], i) => (
            <div key={clipId} className="flex items-center gap-3 text-sm">
              <span className="text-zinc-400 w-16 shrink-0">Part {i + 1}</span>
              <div className="flex-1 h-1.5 bg-zinc-800 rounded-full overflow-hidden">
                <div
                  className="h-full bg-brand rounded-full transition-all duration-300"
                  style={{ width: `${Math.round(pct)}%` }}
                />
              </div>
              <span className="text-zinc-500 w-10 text-right tabular-nums">{Math.round(pct)}%</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
