// apps/web/src/components/LogsProvider.tsx
'use client';

import { useAppStore } from '@/store/appStore';
import { useLogsSSE } from '@/hooks/useLogsSSE';
import { LogPanel } from '@/components/LogPanel';

export function LogsProvider() {
  const goUrl = useAppStore((s) => s.goUrl);
  useLogsSSE(goUrl);
  return <LogPanel />;
}
