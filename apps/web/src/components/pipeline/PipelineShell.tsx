'use client';

import type { ReactNode } from 'react';
import { StepRail } from './StepRail';
import { ContextPanel } from './ContextPanel';
import { usePipelineStore } from '@/store/pipelineStore';

interface PipelineShellProps {
  children: ReactNode;
}

export function PipelineShell({ children }: PipelineShellProps) {
  const run = usePipelineStore((s) => s.run);

  return (
    <div className="flex h-full w-full overflow-hidden">
      {/* Left: step rail — 220px */}
      <div className="w-[220px] shrink-0 border-r border-border bg-background/60 overflow-y-auto">
        <StepRail state={run?.state ?? 'WAITING_URL'} />
      </div>

      {/* Center: active step content */}
      <div className="flex-1 overflow-y-auto p-6">
        {children}
      </div>

      {/* Right: context panel — 300px */}
      <div className="w-[300px] shrink-0 border-l border-border bg-background/60 overflow-y-auto">
        <ContextPanel />
      </div>
    </div>
  );
}
