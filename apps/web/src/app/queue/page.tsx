'use client';

import { useEffect, useState } from 'react';
import { useQueueStore } from '@/store/queueStore';
import { useChannelStore } from '@/store/channelStore';
import { useAppStore } from '@/store/appStore';
import { QueueItemCard } from '@/components/queue/QueueItemCard';
import { ReviewScheduleModal } from '@/components/queue/ReviewScheduleModal';
import { Button } from '@/components/ui/button';
import { createLogger } from '@/lib/logger';

const log = createLogger('QueuePage');

export default function QueuePage() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { items, loading, error, fetchQueue, bulkSchedule } = useQueueStore();
  const { channels, fetchChannels } = useChannelStore();
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [bulkStartAt, setBulkStartAt] = useState('');
  const [bulkInterval, setBulkInterval] = useState(1440);
  const [reviewOpen, setReviewOpen] = useState(false);

  useEffect(() => {
    void fetchQueue();
    if (channels.length === 0) void fetchChannels(goUrl);
  }, [goUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  // Unique channel IDs that appear in the queue
  const channelIdsInQueue = [...new Set(items.map((i) => i.channel_id))];
  const channelOptions = channelIdsInQueue.map((cid) => ({
    id: cid,
    name: channels.find((c) => c.ID === cid)?.ChannelTitle ?? `Channel ${cid}`,
  }));

  const visibleItems = selectedChannelId
    ? items.filter((i) => i.channel_id === selectedChannelId)
    : items;

  const handleBulkConfirm = async () => {
    try {
      await bulkSchedule(new Date(bulkStartAt).toISOString(), bulkInterval);
      setReviewOpen(false);
      setBulkOpen(false);
      setBulkStartAt('');
    } catch (err) {
      log.error('bulk schedule failed', { err });
    }
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Upload Queue</h1>
          <p className="text-sm text-zinc-500 mt-1">
            {items.length} item{items.length !== 1 ? 's' : ''} in queue
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => setBulkOpen((v) => !v)}>
          Bulk Schedule
        </Button>
      </div>

      {/* Review modal (bulk) */}
      <ReviewScheduleModal
        open={reviewOpen}
        mode="bulk"
        items={items}
        startAt={bulkStartAt ? new Date(bulkStartAt).toISOString() : undefined}
        intervalMinutes={bulkInterval}
        channels={channelOptions}
        goUrl={goUrl}
        onConfirm={handleBulkConfirm}
        onCancel={() => setReviewOpen(false)}
      />

      {/* Bulk schedule form */}
      {bulkOpen && (
        <div className="rounded-xl border border-border bg-card p-4 flex flex-wrap items-end gap-3">
          <div className="flex flex-col gap-1">
            <label className="text-xs text-zinc-400">Start at</label>
            <input
              type="datetime-local"
              value={bulkStartAt}
              onChange={(e) => setBulkStartAt(e.target.value)}
              className="rounded-lg bg-[#1A1A26] border border-border px-2.5 py-1.5 text-xs text-[#F0F0F8] focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-zinc-400">Interval (min)</label>
            <input
              type="number"
              min={1}
              value={bulkInterval}
              onChange={(e) => setBulkInterval(Number(e.target.value))}
              className="w-24 rounded-lg bg-[#1A1A26] border border-border px-2.5 py-1.5 text-xs text-[#F0F0F8] focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
          <Button
            size="sm"
            disabled={!bulkStartAt}
            onClick={() => setReviewOpen(true)}
          >
            Preview & Schedule
          </Button>
          <Button size="sm" variant="outline" onClick={() => setBulkOpen(false)}>
            Cancel
          </Button>
        </div>
      )}

      {/* Channel filter pills */}
      {channelOptions.length > 1 && (
        <div className="flex items-center gap-2 flex-wrap">
          <button
            onClick={() => setSelectedChannelId(null)}
            className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
              selectedChannelId === null
                ? 'bg-primary text-primary-foreground border-primary'
                : 'border-border text-zinc-400 hover:border-zinc-500'
            }`}
          >
            Todos
          </button>
          {channelOptions.map((c) => (
            <button
              key={c.id}
              onClick={() => setSelectedChannelId(c.id === selectedChannelId ? null : c.id)}
              className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
                selectedChannelId === c.id
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'border-border text-zinc-400 hover:border-zinc-500'
              }`}
            >
              {c.name}
            </button>
          ))}
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="rounded-md border border-red-800 bg-red-950 px-4 py-3 text-xs text-red-400">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && items.length === 0 && (
        <p className="text-sm text-zinc-500">Carregando fila...</p>
      )}

      {/* Empty state */}
      {!loading && visibleItems.length === 0 && (
        <div className="rounded-xl border border-dashed border-zinc-700 p-10 text-center">
          <p className="text-sm text-zinc-500">Nenhum item na fila.</p>
          <p className="text-xs text-zinc-600 mt-1">
            Confirme um upload no pipeline para adicionar clips aqui.
          </p>
        </div>
      )}

      {/* Queue items */}
      <div className="space-y-3">
        {visibleItems.map((item) => {
          const channelName = channels.find((c) => c.ID === item.channel_id)?.ChannelTitle;
          return (
            <QueueItemCard
              key={item.id}
              item={item}
              channelName={channelName}
              goUrl={goUrl}
            />
          );
        })}
      </div>
    </div>
  );
}
