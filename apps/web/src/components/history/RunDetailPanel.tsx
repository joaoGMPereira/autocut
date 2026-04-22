'use client';

import { createLogger } from '@/lib/logger';
import { useHistoryStore, calcDuration } from '@/store/historyStore';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from '@/components/ui/sheet';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { StepRow } from '@/components/history/StepRow';

const log = createLogger('RunDetailPanel');

const STATUS_CLASSES: Record<string, string> = {
  done: 'text-emerald-400 border-emerald-500/30 bg-emerald-500/5',
  error: 'text-red-400 border-red-400/30 bg-red-400/5',
  running: 'text-blue-400 border-blue-500/30 bg-blue-500/5 animate-pulse',
  pending: 'text-zinc-400 border-zinc-600 bg-zinc-800/50',
};

function truncateUrl(url: string): string {
  try {
    const u = new URL(url);
    const v = u.searchParams.get('v');
    if (v) return `youtube.com/watch?v=${v}`;
    return u.hostname + u.pathname.slice(0, 40);
  } catch {
    return url.slice(0, 60);
  }
}

export function RunDetailPanel() {
  const selectedRun = useHistoryStore((s) => s.selectedRun);
  const clearSelectedRun = useHistoryStore((s) => s.clearSelectedRun);
  const exportCSV = useHistoryStore((s) => s.exportCSV);

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      log.info('closing run detail panel');
      clearSelectedRun();
    }
  };

  const handleExportCSV = () => {
    log.info('exporting CSV from detail panel');
    exportCSV();
  };

  return (
    <Sheet open={selectedRun !== null} onOpenChange={handleOpenChange}>
      <SheetContent side="right" className="sm:max-w-lg w-full flex flex-col gap-0 p-0">
        {selectedRun && (
          <>
            <SheetHeader className="p-4 border-b border-border">
              <SheetTitle className="text-sm font-mono text-zinc-400 truncate">
                #{selectedRun.id}
              </SheetTitle>
              <p className="text-heading text-sm font-medium truncate mt-1">
                {truncateUrl(selectedRun.url)}
              </p>

              <div className="flex items-center gap-2 mt-2 flex-wrap">
                <span
                  className={`rounded-md border px-2 py-0.5 text-xs ${
                    STATUS_CLASSES[selectedRun.status] ?? STATUS_CLASSES.pending
                  }`}
                >
                  {selectedRun.status}
                </span>
                <span className="text-xs text-zinc-500">{selectedRun.mode}</span>
                <span className="text-xs text-zinc-500">
                  {calcDuration(selectedRun.started_at, selectedRun.finished_at)}
                </span>
                {selectedRun.channel_id?.Valid && (
                  <span className="text-xs text-zinc-600">ch#{selectedRun.channel_id.Int64}</span>
                )}
              </div>

              {selectedRun.status === 'error' && (
                <p className="text-xs text-red-400 mt-1">
                  One or more steps failed — see step details below.
                </p>
              )}
            </SheetHeader>

            <ScrollArea className="flex-1 px-4 py-3">
              {selectedRun.steps.length === 0 ? (
                <p className="text-sm text-zinc-500">No step data available.</p>
              ) : (
                <div className="flex flex-col gap-2">
                  {selectedRun.steps.map((step, idx) => (
                    <StepRow key={`${step.step_name}-${idx}`} step={step} />
                  ))}
                </div>
              )}
            </ScrollArea>

            <SheetFooter className="p-4 border-t border-border">
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={handleExportCSV}
              >
                Export visible runs as CSV
              </Button>
            </SheetFooter>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
