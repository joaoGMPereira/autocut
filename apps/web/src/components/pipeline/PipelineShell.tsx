'use client';

import { useState } from 'react';
import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { StepRail } from './StepRail';
import { ContextPanel } from './ContextPanel';
import { BackConfirmDialog } from './BackConfirmDialog';
import { usePipelineStore, isComputeState, isGateState } from '@/store/pipelineStore';
import { useAppStore } from '@/store/appStore';
import { useBackNavigation } from '@/hooks/useBackNavigation';

interface PipelineShellProps {
  children: ReactNode;
}

export function PipelineShell({ children }: PipelineShellProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const run = usePipelineStore((s) => s.run);
  const pendingMode = usePipelineStore((s) => s.pendingMode);
  const cancelRun = usePipelineStore((s) => s.cancelRun);
  const clearRun = usePipelineStore((s) => s.clearRun);
  const activeRunId = usePipelineStore((s) => s.activeRunId);

  const [showCancelDialog, setShowCancelDialog] = useState(false);
  const [showStartOverDialog, setShowStartOverDialog] = useState(false);

  const { showLeaveDialog, setShowLeaveDialog } = useBackNavigation(
    activeRunId,
    run?.state ?? null,
  );

  const displayedState = usePipelineStore((s) => s.displayedState());
  const navIndex = usePipelineStore((s) => s.navIndex);
  const navigateBack = usePipelineStore((s) => s.navigateBack);

  const currentState = run?.state ?? 'WAITING_URL';
  const isCompute = isComputeState(currentState);
  // Back button on gate steps (except WAITING_URL) — purely local nav, no API call
  const canGateBack = isGateState(displayedState) && displayedState !== 'WAITING_URL' && navIndex > 0;
  // Start over button on WAITING_URL — resets entire pipeline to fresh state
  const canUrlStartOver = displayedState === 'WAITING_URL' && run !== null;

  const handleGateBack = () => {
    console.log('[PipelineShell] gateBack triggered', displayedState);
    navigateBack();
  };

  const handleComputeBack = () => {
    console.log('[PipelineShell] computeStateBack triggered', currentState);
    setShowCancelDialog(true);
  };

  const handleCancelConfirm = () => {
    if (activeRunId !== null) {
      void cancelRun(goUrl, activeRunId).then(() => clearRun());
    } else {
      clearRun();
    }
  };

  const handleStartOver = () => {
    setShowStartOverDialog(true);
  };

  const handleStartOverConfirm = () => {
    if (activeRunId !== null) {
      void cancelRun(goUrl, activeRunId).then(() => clearRun());
    } else {
      clearRun();
    }
  };

  const handleLeaveConfirm = () => {
    console.log('[PipelineShell] leavePageConfirmed');
    if (activeRunId !== null) {
      void cancelRun(goUrl, activeRunId).then(() => {
        clearRun();
        window.history.back();
      });
    } else {
      clearRun();
      window.history.back();
    }
  };

  return (
    <div className="flex h-full w-full overflow-hidden">
      {/* Left: step rail — 220px */}
      <div className="w-[220px] shrink-0 border-r border-border bg-background/60 overflow-y-auto">
        <StepRail state={displayedState} mode={pendingMode ?? run?.mode ?? 'ai'} />
      </div>

      {/* Center: active step content + sticky action bar */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <div className="flex-1 overflow-y-auto p-6">
          {isCompute && (
            <div className="mb-4 flex">
              <Button variant="outline" size="xs" onClick={handleComputeBack}>
                ← Cancel &amp; Go Back
              </Button>
            </div>
          )}
          {children}
        </div>
        {canGateBack && (
          <div className="shrink-0 border-t border-border bg-background/80 backdrop-blur-sm px-6 py-3 flex items-center">
            <Button variant="outline" onClick={handleGateBack}>
              ← Back
            </Button>
          </div>
        )}
        {canUrlStartOver && (
          <div className="shrink-0 border-t border-border bg-background/80 backdrop-blur-sm px-6 py-3 flex items-center">
            <Button variant="outline" onClick={handleStartOver}>
              ← Start Over
            </Button>
          </div>
        )}
      </div>

      {/* Right: context panel — 300px */}
      <div className="w-[300px] shrink-0 border-l border-border bg-background/60 overflow-y-auto">
        <ContextPanel />
      </div>

      <BackConfirmDialog
        open={showCancelDialog}
        onOpenChange={setShowCancelDialog}
        variant="cancel-operation"
        onConfirm={handleCancelConfirm}
      />

      <BackConfirmDialog
        open={showLeaveDialog}
        onOpenChange={setShowLeaveDialog}
        variant="leave-page"
        onConfirm={handleLeaveConfirm}
      />

      <BackConfirmDialog
        open={showStartOverDialog}
        onOpenChange={setShowStartOverDialog}
        variant="start-over"
        onConfirm={handleStartOverConfirm}
      />
    </div>
  );
}
