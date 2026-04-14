'use client';

import { useState, useEffect } from 'react';
import type { RunState } from '@/types/pipeline';
import { isTerminalState } from '@/store/pipelineStore';

export function useBackNavigation(activeRunId: number | null, currentState: RunState | null) {
  const [showLeaveDialog, setShowLeaveDialog] = useState(false);

  useEffect(() => {
    const isActive =
      activeRunId !== null &&
      currentState !== null &&
      !isTerminalState(currentState);

    if (!isActive) return;

    // Push sentinel to intercept the next back gesture
    window.history.pushState(null, '', window.location.href);

    const handlePopState = () => {
      console.log('[useBackNavigation] popstate intercepted', activeRunId);
      // Re-push sentinel to prevent navigation
      window.history.pushState(null, '', window.location.href);
      setShowLeaveDialog(true);
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, [activeRunId, currentState]);

  return { showLeaveDialog, setShowLeaveDialog };
}
