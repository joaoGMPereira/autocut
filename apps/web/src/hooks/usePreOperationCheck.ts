'use client';

import { useCallback } from 'react';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import type { ToolStatus } from '@/types/setup';

export interface PreOperationResult {
  ok: boolean;
  missing: ToolStatus[];
}

/**
 * Returns a `validate()` function to call before any processing operation
 * (cut, download, shorts, optimize, etc.).
 *
 * It re-fetches /api/setup/status so the backend re-resolves binary paths,
 * confirming every required tool is still found at its expected location.
 *
 * Usage:
 *   const { validate } = usePreOperationCheck()
 *   const { ok, missing } = await validate()
 *   if (!ok) { show error; return }
 *   // proceed with operation
 */
export function usePreOperationCheck() {
  const goUrl = useAppStore((s) => s.goUrl);
  const fetchStatus = useSetupStore((s) => s.fetchStatus);
  const missingRequired = useSetupStore((s) => s.missingRequired);

  const validate = useCallback(async (): Promise<PreOperationResult> => {
    await fetchStatus(goUrl);
    const missing = missingRequired();
    return { ok: missing.length === 0, missing };
  }, [goUrl, fetchStatus, missingRequired]);

  return { validate };
}
