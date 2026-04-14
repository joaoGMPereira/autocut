import type { GatePayload } from '@/types/pipeline';

interface StepReviewClipsProps {
  historical?: GatePayload;
}

export function StepReviewClips({ historical }: StepReviewClipsProps) {
  const isHistorical = historical !== undefined;

  return (
    <div className="space-y-2 max-w-lg">
      <h2 className="text-xl font-semibold text-foreground">Review Clips</h2>
      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}
      <p className="text-sm text-zinc-500">TODO: FP-023–024 — clip selection for upload</p>
    </div>
  );
}
