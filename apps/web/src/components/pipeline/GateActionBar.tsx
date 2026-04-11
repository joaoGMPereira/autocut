'use client';

import type { ReactNode } from 'react';
import { usePipelineStore, isGateState } from '@/store/pipelineStore';

interface GateActionBarProps {
  children: ReactNode;
}

export function GateActionBar({ children }: GateActionBarProps) {
  const run = usePipelineStore((s) => s.run);

  // Allow render when run === null (initial WAITING_URL step, no run created yet)
  // or when an active run is in a gate state awaiting user decision.
  if (run !== null && !isGateState(run.state)) return null;

  return (
    <div className="sticky bottom-0 z-10 border-t border-border bg-background/80 backdrop-blur-sm px-6 py-3 flex items-center justify-end gap-3">
      {children}
    </div>
  );
}
