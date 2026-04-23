'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { SectionPanel } from '@/components/ui/section-panel';

export function ChannelInfoCard() {
  const run = usePipelineStore((s) => s.run);

  if (!run?.channel_id) return null;

  return (
    <SectionPanel size="sm" title="Channel">
      <p className="text-foreground font-medium">#{run.channel_id}</p>
    </SectionPanel>
  );
}
