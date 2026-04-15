'use client';

import { useEffect, useRef } from 'react';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import type { ToolStatus } from '@/types/setup';
import { AUTO_INSTALL_TOOLS } from '@/types/setup';

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
  const fetchHardware = useSetupStore((s) => s.fetchHardware);
  const allRequiredInstalled = useSetupStore((s) => s.allRequiredInstalled);
  const missingRequired = useSetupStore((s) => s.missingRequired);
  const checkUpdate = useSetupStore((s) => s.checkUpdate);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // Tracks whether we've already fired update checks this session
  const updateCheckedRef = useRef(false);

  // Initial fetch on mount
  useEffect(() => {
    fetchStatus(goUrl);
    fetchHardware(goUrl);
  }, [goUrl, fetchStatus, fetchHardware]);

  // Once tools are confirmed installed, fire update checks (one-time per session)
  useEffect(() => {
    if (loading || updateCheckedRef.current) return;
    if (!allRequiredInstalled()) return;

    updateCheckedRef.current = true;
    // Fire-and-forget: non-blocking, won't delay app startup
    for (const tool of AUTO_INSTALL_TOOLS) {
      const t = tools.find((x) => x.name === tool);
      if (t?.installed) {
        checkUpdate(goUrl, tool);
      }
    }
  }, [loading, tools, goUrl, checkUpdate, allRequiredInstalled]);

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
