import type { GatePayload } from '@/types/pipeline';

interface StepReviewHighlightsProps {
  historical?: GatePayload;
}

export function StepReviewHighlights({ historical }: StepReviewHighlightsProps) {
  const isHistorical = historical !== undefined;

  return (
    <div className="space-y-2 max-w-lg">
      <h2 className="text-xl font-semibold text-foreground">Review Highlights</h2>
      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}
      <p className="text-sm text-zinc-500">TODO: FP-019–022 — highlight detection review and selection</p>
    </div>
  );
}
