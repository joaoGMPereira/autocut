import type { GatePayload } from '@/types/pipeline';

interface StepReviewMetadataProps {
  historical?: GatePayload;
}

export function StepReviewMetadata({ historical }: StepReviewMetadataProps) {
  const isHistorical = historical !== undefined;

  return (
    <div className="space-y-2 max-w-lg">
      <h2 className="text-xl font-semibold text-foreground">Review Metadata</h2>
      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}
      <p className="text-sm text-zinc-500">TODO: FP-061–070 — title, description, tags per clip</p>
    </div>
  );
}
