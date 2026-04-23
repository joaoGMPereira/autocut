'use client';

import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import type { JobStatus } from '@/store/effectsStore';

interface SseLogPanelProps {
  logs: string[];
  status: JobStatus;
}

const STATUS_BADGE: Record<JobStatus, { label: string; variant: 'secondary' | 'info' | 'success' | 'destructive'; extra?: string }> = {
  idle: { label: 'Idle', variant: 'secondary' },
  running: { label: 'Running…', variant: 'info', extra: 'animate-pulse' },
  done: { label: 'Done', variant: 'success' },
  error: { label: 'Error', variant: 'destructive' },
};

export function SseLogPanel({ logs, status }: SseLogPanelProps) {
  if (status === 'idle' && logs.length === 0) return null;

  const badge = STATUS_BADGE[status];

  return (
    <div className="mt-4 rounded-lg border border-border bg-background/50 p-3 flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-subtle font-medium">Job Log</span>
        <Badge variant={badge.variant} className={badge.extra}>
          {badge.label}
        </Badge>
      </div>
      <ScrollArea className="h-36 w-full">
        <div className="flex flex-col gap-0.5 font-mono text-xs text-subtle pr-2">
          {logs.length === 0 ? (
            <span className="text-caption italic">Waiting for output…</span>
          ) : (
            logs.map((line, i) => (
              <span key={i} className={line.startsWith('Error') ? 'text-destructive' : ''}>
                {line}
              </span>
            ))
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
