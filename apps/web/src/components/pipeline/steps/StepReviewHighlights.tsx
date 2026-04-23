import type { GatePayload } from '@/types/pipeline';

interface StepReviewHighlightsProps {
  historical?: GatePayload;
}

export function StepReviewHighlights({ historical }: StepReviewHighlightsProps) {
  const isHistorical = historical !== undefined;

  return (
    <div className="space-y-2 max-w-lg">
      <h2 className="text-xl font-semibold text-heading">Review Highlights</h2>
      {isHistorical && (
        <div className="rounded-md border border-border bg-card px-4 py-3 text-xs text-prose">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}
      <p className="text-sm text-subtle">TODO: FP-019–022 — highlight detection review and selection</p>
    </div>
  );
}
