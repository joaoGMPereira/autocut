'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { ChannelInfoCard } from './ChannelInfoCard';
import { DownloadInfoCard } from './DownloadInfoCard';
import { LogStream } from './LogStream';

export function ContextPanel() {
  const run = usePipelineStore((s) => s.run);

  return (
    <div className="sticky top-0 p-4 space-y-3">
      <p className="text-xs font-semibold uppercase tracking-wider text-zinc-500 pb-1">Info</p>

      {run && (
        <div className="rounded-lg border border-border bg-card p-3 text-xs space-y-1">
          <p className="text-zinc-500">Run ID: <span className="text-foreground">{run.id}</span></p>
          <p className="text-zinc-500">Mode: <span className="text-foreground">{run.mode || '—'}</span></p>
          <p className="text-zinc-500">State: <span className="text-foreground">{run.state}</span></p>
        </div>
      )}

      <ChannelInfoCard />
      <DownloadInfoCard />
      <LogStream />
    </div>
  );
}
