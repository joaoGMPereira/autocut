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
    <div className="rounded-lg border border-border bg-surface-inset p-3 font-mono text-xs h-40 overflow-y-auto">
      {run.state && (
        <p className="text-prose">
          <span className="text-caption">[state] </span>
          {run.state}
        </p>
      )}
      {run.active_phase && (
        <p className="text-prose">
          <span className="text-caption">[phase] </span>
          {run.active_phase}
        </p>
      )}
      {phaseProgress && (
        <p className="text-brand">
          <span className="text-caption">[progress] </span>
          {phaseProgress.phase} {Math.round(phaseProgress.percentDone)}%
          {phaseProgress.speedKbs != null ? ` @ ${phaseProgress.speedKbs.toFixed(0)} KB/s` : ''}
        </p>
      )}
      {run.error && (
        <p className="text-destructive">
          <span className="text-caption">[error] </span>
          {run.error}
        </p>
      )}
      <div ref={bottomRef} />
    </div>
  );
}
