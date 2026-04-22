'use client';

import { useState } from 'react';
import { Trash2, Calendar, RotateCcw, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import type { QueueItem } from '@/store/queueStore';
import { useQueueStore } from '@/store/queueStore';
import { ReviewScheduleModal } from '@/components/queue/ReviewScheduleModal';
import { createLogger } from '@/lib/logger';

const log = createLogger('QueueItemCard');

interface Props {
  item: QueueItem;
  channelName?: string;
  goUrl: string;
}

function statusVariant(status: QueueItem['status']): 'secondary' | 'info' | 'success' | 'destructive' {
  switch (status) {
    case 'queued':   return 'secondary';
    case 'running':  return 'info';
    case 'uploaded': return 'success';
    case 'error':    return 'destructive';
  }
}

export function QueueItemCard({ item, channelName, goUrl }: Props) {
  const { retryItem, deleteItem, scheduleItem } = useQueueStore();
  const [reviewOpen, setReviewOpen] = useState(false);
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

  const handleScheduleConfirm = async (publishAt?: string) => {
    if (!publishAt) return;
    setActionLoading('schedule');
    try {
      await scheduleItem(item.id, publishAt);
      setReviewOpen(false);
    } catch (err) {
      log.error('schedule failed', { id: item.id, err });
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <>
      <ReviewScheduleModal
        open={reviewOpen}
        mode="individual"
        item={item}
        channelName={channelName}
        goUrl={goUrl}
        onConfirm={handleScheduleConfirm}
        onCancel={() => setReviewOpen(false)}
      />
    <div className="rounded-xl bg-card border border-border p-4 flex gap-3 items-start">
      {/* Thumbnail */}
      <div className="shrink-0 w-20 rounded-lg overflow-hidden bg-muted aspect-video flex items-center justify-center">
        <img
          src={`${goUrl}/api/queue/${item.id}/thumbnail`}
          alt="thumbnail"
          className="w-full h-full object-cover"
          onError={(e) => {
            (e.currentTarget as HTMLImageElement).style.display = 'none';
          }}
        />
      </div>

      {/* Main content */}
      <div className="flex-1 min-w-0 flex flex-col gap-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-semibold text-heading truncate">
            {item.title ?? 'Untitled'}
          </span>
          <Badge variant={statusVariant(item.status)}>{item.status}</Badge>
          <span className="text-[11px] text-subtle font-mono">{item.video_type}</span>
        </div>

        {/* Channel badge */}
        {channelName && (
          <Badge variant="outline" className="w-fit text-[10px]">
            {channelName}
          </Badge>
        )}

        {item.publish_at && (
          <span className="inline-flex w-fit items-center gap-1 rounded-full bg-violet-500/15 border border-violet-500/30 px-2 py-0.5 text-[11px] font-medium text-violet-400">
            <Calendar className="size-3" />
            Scheduled: {new Date(item.publish_at).toLocaleString()}
          </span>
        )}

        {item.status === 'error' && item.error && (
          <p className="text-xs text-destructive bg-destructive/10 rounded-md px-3 py-1.5 border border-destructive/20">
            {item.error}
          </p>
        )}

        {item.status === 'uploaded' && item.youtube_url && (
          <a
            href={item.youtube_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-xs text-brand hover:underline w-fit"
          >
            <ExternalLink className="size-3" />
            View on YouTube
          </a>
        )}

      </div>

      {/* Action buttons */}
      <TooltipProvider>
        <div className="flex items-center gap-1.5 shrink-0">
          {item.status === 'error' && (
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
                onClick={() => setReviewOpen(true)}
              >
                <Calendar className="size-3.5" />
                <span className="sr-only">Schedule</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Review & schedule</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-7 w-7 p-0 text-destructive hover:text-destructive hover:border-destructive/50"
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
    </>
  );
}
