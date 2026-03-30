'use client';

import { useEffect, useRef } from 'react';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import type { ToolStatus } from '@/types/setup';

interface UseSetupCheckResult {
  isReady: boolean;
  isLoading: boolean;
  tools: ToolStatus[];
  missingRequired: ToolStatus[];
  error: string | null;
}

export function useSetupCheck(): UseSetupCheckResult {
  const goUrl = useAppStore((s) => s.goUrl);
  const tools = useSetupStore((s) => s.tools);
  const loading = useSetupStore((s) => s.loading);
  const error = useSetupStore((s) => s.error);
  const fetchStatus = useSetupStore((s) => s.fetchStatus);
  const allRequiredInstalled = useSetupStore((s) => s.allRequiredInstalled);
  const missingRequired = useSetupStore((s) => s.missingRequired);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Initial fetch on mount
  useEffect(() => {
    fetchStatus(goUrl);
  }, [goUrl, fetchStatus]);

  // Re-poll every 30s while not all required tools are installed
  useEffect(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }

    if (!loading && !allRequiredInstalled()) {
      pollingRef.current = setInterval(() => {
        fetchStatus(goUrl);
      }, 30_000);
    }

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
      }
    };
  }, [loading, goUrl, fetchStatus, allRequiredInstalled]);

  return {
    isReady: allRequiredInstalled(),
    isLoading: loading,
    tools,
    missingRequired: missingRequired(),
    error,
  };
}
