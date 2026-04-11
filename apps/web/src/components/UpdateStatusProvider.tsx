'use client';

import { useEffect } from 'react';
import type { UpdateProgress } from '@autocut/shared';
import { useAppStore } from '@/store/appStore';
import { useUpdateDispatcher } from '@/hooks/useUpdateDispatcher';

/**
 * Subscribes to Electron IPC update status events and feeds them into the
 * global appStore. Also runs useUpdateDispatcher so the dispatcherStore
 * always reflects current update state — even outside the pipeline wizard.
 *
 * Must be a 'use client' component since layout.tsx is a Server Component.
 */
export function UpdateStatusProvider({ children }: { children: React.ReactNode }) {
  const setUpdateStatus = useAppStore((s) => s.setUpdateStatus);

  useEffect(() => {
    const api = (window as unknown as Record<string, unknown>).electronAPI as
      | {
          getUpdateStatus?: () => Promise<UpdateProgress>;
          onUpdateStatus?: (cb: (s: UpdateProgress) => void) => void;
        }
      | undefined;

    if (!api) return;

    // Hydrate current status (app may have started checking before this mounts)
    api.getUpdateStatus?.().then(setUpdateStatus).catch(() => undefined);

    // Subscribe to live changes from the main process
    api.onUpdateStatus?.(setUpdateStatus);
  }, [setUpdateStatus]);

  // Bridge update status into the dispatcher store globally
  useUpdateDispatcher();

  return <>{children}</>;
}
