'use client';

import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import type { JobStatus } from '@/store/effectsStore';

interface SseLogPanelProps {
  logs: string[];
  status: JobStatus;
}

const STATUS_BADGE: Record<JobStatus, { label: string; className: string }> = {
  idle: { label: 'Idle', className: 'bg-zinc-800 text-zinc-400 border-zinc-700' },
  running: { label: 'Running…', className: 'bg-blue-500/10 text-blue-400 border-blue-500/30 animate-pulse' },
  done: { label: 'Done', className: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30' },
  error: { label: 'Error', className: 'bg-red-500/10 text-red-400 border-red-500/30' },
};

export function SseLogPanel({ logs, status }: SseLogPanelProps) {
  if (status === 'idle' && logs.length === 0) return null;

  const badge = STATUS_BADGE[status];

  return (
    <div className="mt-4 rounded-lg border border-border bg-zinc-900/50 p-3 flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-zinc-500 font-medium">Job Log</span>
        <Badge variant="outline" className={badge.className}>
          {badge.label}
        </Badge>
      </div>
      <ScrollArea className="h-36 w-full">
        <div className="flex flex-col gap-0.5 font-mono text-xs text-zinc-400 pr-2">
          {logs.length === 0 ? (
            <span className="text-zinc-600 italic">Waiting for output…</span>
          ) : (
            logs.map((line, i) => (
              <span key={i} className={line.startsWith('Error') ? 'text-red-400' : ''}>
                {line}
              </span>
            ))
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
