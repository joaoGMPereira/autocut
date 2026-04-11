'use client';

import { usePipelineStore } from '@/store/pipelineStore';

export function ChannelInfoCard() {
  const run = usePipelineStore((s) => s.run);

  if (!run?.channel_id) return null;

  return (
    <div className="rounded-lg border border-border bg-card p-3 text-sm">
      <p className="text-xs font-semibold uppercase tracking-wider text-zinc-500 mb-1">Channel</p>
      <p className="text-foreground font-medium">#{run.channel_id}</p>
    </div>
  );
}
