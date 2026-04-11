'use client';

import { useEffect, type ReactNode } from 'react';
import { createLogger } from '@/lib/logger';
import { useHistoryStore } from '@/store/historyStore';
import { useAppStore } from '@/store/appStore';

const log = createLogger('HistoryShell');

interface HistoryShellProps {
  children: ReactNode;
}

export function HistoryShell({ children }: HistoryShellProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const fetchRuns = useHistoryStore((s) => s.fetchRuns);

  useEffect(() => {
    if (goUrl) {
      log.info('initial fetch on mount');
      void fetchRuns(goUrl);
    }
  }, [goUrl, fetchRuns]);

  return <>{children}</>;
}
