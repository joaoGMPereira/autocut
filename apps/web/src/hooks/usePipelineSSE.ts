'use client';

import { useEffect, useRef } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('usePipelineSSE');

/**
 * Subscribes to the SSE stream for the given run ID.
 * Automatically re-subscribes when runId changes and cleans up on unmount.
 */
export function usePipelineSSE(goUrl: string, runId: number | null) {
  const subscribeSSE = usePipelineStore((s) => s.subscribeSSE);
  const stopSSE = usePipelineStore((s) => s.stopSSE);
  const cleanupRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!runId) return;
    log.info('[usePipelineSSE] subscribing', { runId });
    // Stop any previous subscription
    if (cleanupRef.current) {
      cleanupRef.current();
      cleanupRef.current = null;
    }
    cleanupRef.current = subscribeSSE(goUrl, runId);
    return () => {
      if (cleanupRef.current) {
        log.info('[usePipelineSSE] unsubscribing', { runId });
        cleanupRef.current();
        cleanupRef.current = null;
      }
    };
  }, [goUrl, runId, subscribeSSE]);

  return { stopSSE };
}
