'use client';

import { usePipelineStore } from '@/store/pipelineStore';

export function StepGeneratingClipsLongform() {
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);

  return (
    <div className="space-y-2">
      <h2 className="text-xl font-semibold">Cutting Clips</h2>
      <p className="text-sm text-zinc-500">Cutting video into segments…</p>
      {phaseProgress && (
        <div className="mt-4">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-zinc-400">{phaseProgress.phase}</span>
            <span className="text-xs text-zinc-500">{Math.round(phaseProgress.percentDone)}%</span>
          </div>
          <div className="w-full h-1 bg-zinc-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-cyan-400 rounded-full transition-all duration-500"
              style={{ width: `${phaseProgress.percentDone}%` }}
            />
          </div>
        </div>
      )}
    </div>
  );
}
