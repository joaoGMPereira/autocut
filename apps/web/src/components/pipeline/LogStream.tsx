'use client';

import { useEffect, useRef } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';

export function LogStream() {
  const run = usePipelineStore((s) => s.run);
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [phaseProgress]);

  if (!run) return null;

  return (
    <div className="rounded-lg border border-border bg-zinc-950 p-3 font-mono text-xs h-40 overflow-y-auto">
      {run.state && (
        <p className="text-zinc-400">
          <span className="text-zinc-600">[state] </span>
          {run.state}
        </p>
      )}
      {run.active_phase && (
        <p className="text-zinc-400">
          <span className="text-zinc-600">[phase] </span>
          {run.active_phase}
        </p>
      )}
      {phaseProgress && (
        <p className="text-brand">
          <span className="text-zinc-600">[progress] </span>
          {phaseProgress.phase} {Math.round(phaseProgress.percentDone)}%
          {phaseProgress.speedKbs != null ? ` @ ${phaseProgress.speedKbs.toFixed(0)} KB/s` : ''}
        </p>
      )}
      {run.error && (
        <p className="text-red-400">
          <span className="text-zinc-600">[error] </span>
          {run.error}
        </p>
      )}
      <div ref={bottomRef} />
    </div>
  );
}
