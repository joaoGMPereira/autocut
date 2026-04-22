'use client';

import type { ExecutePhase } from '@/types/pipeline';

interface SubStepProgressProps {
  phase: ExecutePhase;
  label: string;
  icon?: string;
  percentDone: number;
  speedKbs?: number;
  etaSec?: number;
  isActive?: boolean;
}

export function SubStepProgress({
  label,
  icon,
  percentDone,
  speedKbs,
  etaSec,
  isActive,
}: SubStepProgressProps) {
  const pct = Math.min(100, Math.round(percentDone));
  const isDone = pct >= 100;

  const speed = speedKbs != null
    ? speedKbs > 1024
      ? `${(speedKbs / 1024).toFixed(1)} MB/s`
      : `${speedKbs.toFixed(0)} KB/s`
    : null;

  const eta = etaSec != null
    ? etaSec > 60
      ? `${Math.floor(etaSec / 60)}m ${etaSec % 60}s`
      : `${etaSec}s`
    : null;

  return (
    <div className={`flex items-center gap-3 py-2 px-3 rounded-lg transition-colors ${isActive ? 'bg-zinc-800/60' : ''}`}>
      {icon && <span className="text-base w-5 text-center">{icon}</span>}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <span className={`text-sm truncate ${isActive ? 'text-foreground' : isDone ? 'text-emerald-400' : 'text-zinc-500'}`}>
            {label}
          </span>
          <span className="text-xs text-zinc-500 ml-2">{pct}%</span>
        </div>
        <div className="w-full h-1 bg-zinc-700 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-500 ${isDone ? 'bg-emerald-400' : isActive ? 'bg-brand' : 'bg-zinc-600'}`}
            style={{ width: `${pct}%` }}
          />
        </div>
        {(speed || eta) && isActive && (
          <div className="flex gap-3 mt-1 text-xs text-zinc-500">
            {speed && <span>{speed}</span>}
            {eta && <span>ETA {eta}</span>}
          </div>
        )}
      </div>
    </div>
  );
}
