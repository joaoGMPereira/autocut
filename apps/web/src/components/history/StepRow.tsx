'use client';

import { createLogger } from '@/lib/logger';
import type { RunStep } from '@/store/historyStore';
import { calcDuration } from '@/store/historyStore';
import { Badge } from '@/components/ui/badge';

const log = createLogger('StepRow');

const STEP_STATUS_VARIANT: Record<string, 'success' | 'destructive' | 'info' | 'secondary'> = {
  done: 'success',
  error: 'destructive',
  running: 'info',
  pending: 'secondary',
};

interface StepRowProps {
  step: RunStep;
}

export function StepRow({ step }: StepRowProps) {
  const duration = calcDuration(step.started_at, step.finished_at);

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
    <div className="flex flex-col gap-1 rounded-lg border border-border bg-background px-3 py-2.5">
      <div className="flex items-center gap-3">
        <span className="flex-1 text-sm font-medium text-heading">{step.step_name}</span>

        <Badge
          variant={STEP_STATUS_VARIANT[step.status] ?? STEP_STATUS_VARIANT.pending}
          className="shrink-0"
        >
          {step.status}
        </Badge>

        <span className="shrink-0 font-mono text-xs text-subtle">{duration}</span>
      </div>

      {outputPreview && (
        <p className="font-mono text-xs text-subtle break-all">{outputPreview}</p>
      )}

      {step.error_message && (
        <p className="text-xs text-destructive break-all">{step.error_message}</p>
      )}
    </div>
  );
}
