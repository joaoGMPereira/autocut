'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { DownloadInfoCard } from '@/components/pipeline/DownloadInfoCard';

export function ContextPanel() {
  const run = usePipelineStore((s) => s.run);

  return (
    <div className="sticky top-0 p-4 space-y-3">
      <p className="text-xs font-semibold uppercase tracking-wider text-subtle pb-1">Info</p>

      {run ? (
        <div className="rounded-lg border border-border bg-card p-3 text-xs space-y-1">
          <p className="text-subtle">Run ID: <span className="text-foreground">{run.id}</span></p>
          <p className="text-subtle">State: <span className="text-foreground">{run.state}</span></p>
        </div>
      ) : (
        <p className="text-xs text-caption">No active run</p>
      )}

      <DownloadInfoCard />
    </div>
  );
}
