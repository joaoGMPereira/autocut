'use client';

import { useState } from 'react';
import { Trash2, Calendar, RotateCcw, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import type { QueueItem } from '@/store/queueStore';
import { useQueueStore } from '@/store/queueStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('QueueItemCard');

interface Props {
  item: QueueItem;
  channelName?: string;
  goUrl: string;
}

function StatusBadge({ status }: { status: QueueItem['status'] }) {
  const variants: Record<QueueItem['status'], string> = {
    queued: 'bg-zinc-500/15 text-zinc-400 border-zinc-500/30',
    running: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
    done: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
    failed: 'bg-red-500/15 text-red-400 border-red-500/30',
  };
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold ${variants[status]}`}
    >
      {status}
    </span>
  );
}

export function QueueItemCard({ item, channelName, goUrl }: Props) {
  const { retryItem, deleteItem, scheduleItem } = useQueueStore();
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [scheduleDraft, setScheduleDraft] = useState('');
  const [actionLoading, setActionLoading] = useState<'retry' | 'delete' | 'schedule' | null>(null);

  const handleRetry = async () => {
    setActionLoading('retry');
    try {
      await retryItem(item.id);
    } catch (err) {
      log.error('retry failed', { id: item.id, err });
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    setActionLoading('delete');
    try {
      await deleteItem(item.id);
    } catch (err) {
      log.error('delete failed', { id: item.id, err });
      setActionLoading(null);
    }
  };

  const handleSchedule = async () => {
    if (!scheduleDraft) return;
    setActionLoading('schedule');
    try {
      await scheduleItem(item.id, new Date(scheduleDraft).toISOString());
      setScheduleOpen(false);
      setScheduleDraft('');
    } catch (err) {
      log.error('schedule failed', { id: item.id, err });
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <div className="rounded-xl bg-card border border-border p-4 flex gap-3 items-start">
      {/* Thumbnail */}
      <div className="shrink-0 w-20 rounded-lg overflow-hidden bg-zinc-800 aspect-video flex items-center justify-center">
        {item.thumbnail_path ? (
          <img
            src={`${goUrl}/api/queue/${item.id}/thumbnail`}
            alt="thumbnail"
            className="w-full h-full object-cover"
          />
        ) : (
          <div className="w-full h-full bg-zinc-700" />
        )}
      </div>

      {/* Main content */}
      <div className="flex-1 min-w-0 flex flex-col gap-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-semibold text-[#F0F0F8] truncate">
            {item.title ?? 'Untitled'}
          </span>
          <StatusBadge status={item.status} />
          <span className="text-[11px] text-[#5C5C80] font-mono">{item.video_type}</span>
        </div>

        {/* Channel badge */}
        {channelName && (
          <span className="inline-flex w-fit items-center rounded-full border border-zinc-700 px-2 py-0.5 text-[10px] text-zinc-500">
            {channelName}
          </span>
        )}

        {item.publish_at && (
          <span className="inline-flex w-fit items-center gap-1 rounded-full bg-violet-500/15 border border-violet-500/30 px-2 py-0.5 text-[11px] font-medium text-violet-400">
            <Calendar className="size-3" />
            Scheduled: {new Date(item.publish_at).toLocaleString()}
          </span>
        )}

        {item.status === 'failed' && item.error && (
          <p className="text-xs text-red-400 bg-red-500/10 rounded-md px-3 py-1.5 border border-red-500/20">
            {item.error}
          </p>
        )}

        {item.status === 'done' && item.youtube_url && (
          <a
            href={item.youtube_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-xs text-[#00D4FF] hover:underline w-fit"
          >
            <ExternalLink className="size-3" />
            View on YouTube
          </a>
        )}

        {/* Inline schedule picker */}
        {scheduleOpen && (
          <div className="flex items-center gap-2 mt-1">
            <input
              type="datetime-local"
              value={scheduleDraft}
              onChange={(e) => setScheduleDraft(e.target.value)}
              className="rounded-lg bg-[#1A1A26] border border-border px-2.5 py-1.5 text-xs text-[#F0F0F8] focus:outline-none focus:border-[#00D4FF]/60"
            />
            <Button
              size="sm"
              className="h-7 text-xs"
              disabled={!scheduleDraft || actionLoading === 'schedule'}
              onClick={() => void handleSchedule()}
            >
              {actionLoading === 'schedule' ? 'Saving…' : 'Set'}
            </Button>
            <Button
              size="sm"
              variant="outline"
              className="h-7 text-xs"
              onClick={() => { setScheduleOpen(false); setScheduleDraft(''); }}
            >
              Cancel
            </Button>
          </div>
        )}
      </div>

      {/* Action buttons */}
      <TooltipProvider>
        <div className="flex items-center gap-1.5 shrink-0">
          {item.status === 'failed' && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 w-7 p-0"
                  disabled={actionLoading === 'retry'}
                  onClick={() => void handleRetry()}
                >
                  <RotateCcw className="size-3.5" />
                  <span className="sr-only">Retry</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Retry</TooltipContent>
            </Tooltip>
          )}

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-7 w-7 p-0"
                onClick={() => setScheduleOpen((v) => !v)}
              >
                <Calendar className="size-3.5" />
                <span className="sr-only">Schedule</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Schedule publish time</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-7 w-7 p-0 text-red-400 hover:text-red-300 hover:border-red-500/50"
                disabled={actionLoading === 'delete'}
                onClick={() => void handleDelete()}
              >
                <Trash2 className="size-3.5" />
                <span className="sr-only">Delete</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Remove from queue</TooltipContent>
          </Tooltip>
        </div>
      </TooltipProvider>
    </div>
  );
}
