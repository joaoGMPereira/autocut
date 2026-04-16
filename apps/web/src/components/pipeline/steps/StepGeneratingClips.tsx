'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { SubStepProgress } from '../SubStepProgress';

export function StepGeneratingClips() {
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);
  const clipProgress = usePipelineStore((s) => s.clipProgress);
  const entries = clipProgress ? Object.entries(clipProgress) : [];

  return (
    <div className="space-y-4 max-w-xl">
      <SubStepProgress
        phase="cut"
        label="Generating clips"
        percentDone={phaseProgress?.phase === 'cut' ? phaseProgress.percentDone : 0}
        isActive={true}
      />
      {entries.length > 0 && (
        <div className="space-y-2 pl-2">
          {entries.map(([clipId, pct], i) => (
            <div key={clipId} className="flex items-center gap-3 text-sm">
              <span className="text-zinc-400 w-16 shrink-0">Part {i + 1}</span>
              <div className="flex-1 h-1.5 bg-zinc-800 rounded-full overflow-hidden">
                <div
                  className="h-full bg-cyan-500 rounded-full transition-all duration-300"
                  style={{ width: `${pct}%` }}
                />
              </div>
              <span className="text-zinc-500 w-10 text-right tabular-nums">{Math.round(pct as number)}%</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
