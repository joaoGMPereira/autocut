'use client';

import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { usePipelineStore, isGateState } from '@/store/pipelineStore';

interface GateActionBarProps {
  children: ReactNode;
}

export function GateActionBar({ children }: GateActionBarProps) {
  const run = usePipelineStore((s) => s.run);
  const displayedState = usePipelineStore((s) => s.displayedState());
  const navigateBack = usePipelineStore((s) => s.navigateBack);
  const navHistory = usePipelineStore((s) => s.navHistory);
  const navIndex = usePipelineStore((s) => s.navIndex);

  // Allow render when run === null (initial WAITING_URL step, no run created yet)
  // or when an active run is in a gate state awaiting user decision.
  if (run !== null && !isGateState(displayedState)) return null;

  // Show Back button on all gate steps except WAITING_URL
  const canGoBack = displayedState !== 'WAITING_URL' && navIndex > 0;

  const handleBack = () => {
    console.log('[GateActionBar] navigateBack triggered', displayedState);
    navigateBack();
  };

  return (
    <div className="sticky bottom-0 z-10 border-t border-border bg-background/80 backdrop-blur-sm px-6 py-3 flex items-center justify-end gap-3">
      {canGoBack && (
        <Button variant="outline" onClick={handleBack}>
          Back
        </Button>
      )}
      {children}
    </div>
  );
}
