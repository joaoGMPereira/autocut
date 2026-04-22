'use client';

import { createLogger } from '@/lib/logger';
import type { RunStep } from '@/store/historyStore';
import { calcDuration } from '@/store/historyStore';

const log = createLogger('StepRow');

const STEP_STATUS_CLASSES: Record<string, string> = {
  done: 'text-emerald-400 border-emerald-500/30 bg-emerald-500/5',
  error: 'text-red-400 border-red-400/30 bg-red-400/5',
  running: 'text-blue-400 border-blue-500/30 bg-blue-500/5',
  pending: 'text-zinc-400 border-zinc-600 bg-zinc-800/50',
};

interface StepRowProps {
  step: RunStep;
}

export function StepRow({ step }: StepRowProps) {
  const duration = calcDuration(step.started_at, step.finished_at);
  const statusClass = STEP_STATUS_CLASSES[step.status] ?? STEP_STATUS_CLASSES.pending;

  let outputPreview = '';
  if (step.output_json != null) {
    try {
      const raw = JSON.stringify(step.output_json);
      outputPreview = raw.length > 120 ? raw.slice(0, 117) + '...' : raw;
    } catch {
      log.error('failed to stringify step output', { step: step.step_name });
    }
  }

  return (
    <div className="flex flex-col gap-1 rounded-lg border border-border bg-zinc-950 px-3 py-2.5">
      <div className="flex items-center gap-3">
        <span className="flex-1 text-sm font-medium text-heading">{step.step_name}</span>

        <span
          className={`shrink-0 rounded-md border px-2 py-0.5 text-xs ${statusClass}`}
        >
          {step.status}
        </span>

        <span className="shrink-0 font-mono text-xs text-zinc-500">{duration}</span>
      </div>

      {outputPreview && (
        <p className="font-mono text-xs text-zinc-500 break-all">{outputPreview}</p>
      )}

      {step.error_message && (
        <p className="text-xs text-red-400 break-all">{step.error_message}</p>
      )}
    </div>
  );
}
