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

type StatusFilter = 'queue' | 'history';

export default function QueuePage() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { items, loading, error, fetchQueue, bulkSchedule, uploadNow } = useQueueStore();
  const { channels, fetchChannels } = useChannelStore();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('queue');
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [bulkStartAt, setBulkStartAt] = useState('');
  const [bulkIntervalDays, setBulkIntervalDays] = useState(1);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [uploadNowLoading, setUploadNowLoading] = useState(false);

  useEffect(() => {
    void fetchQueue();
    if (channels.length === 0) void fetchChannels(goUrl);
  }, [goUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  // Counts for tab badges
  const queueCount = items.filter((i) => ['queued', 'running', 'error'].includes(i.status)).length;
  const historyCount = items.filter((i) => i.status === 'uploaded').length;

  // Filter by status tab
  const statusFiltered = items.filter((i) =>
    statusFilter === 'queue'
      ? ['queued', 'running', 'error'].includes(i.status)
      : i.status === 'uploaded',
  );

  // Channel options derived from status-filtered items
  const channelIdsInView = [...new Set(statusFiltered.map((i) => i.channel_id))];
  const channelOptions = channelIdsInView.map((cid) => ({
    id: cid,
    name: channels.find((c) => c.ID === cid)?.ChannelTitle ?? `Canal ${cid}`,
  }));

  // Filter by selected channel
  const channelFiltered = selectedChannelId
    ? statusFiltered.filter((i) => i.channel_id === selectedChannelId)
    : statusFiltered;

  // Group by channel when "Todos" is selected
  const groups =
    selectedChannelId === null
      ? channelOptions
          .map((co) => ({
            channelId: co.id,
            channelName: co.name,
            items: channelFiltered.filter((i) => i.channel_id === co.id),
          }))
          .filter((g) => g.items.length > 0)
      : [
          {
            channelId: selectedChannelId,
            channelName: channelOptions.find((c) => c.id === selectedChannelId)?.name ?? '',
            items: channelFiltered,
          },
        ];

  const handleStatusFilterChange = (f: StatusFilter) => {
    setStatusFilter(f);
    setSelectedChannelId(null);
  };

  const handleBulkConfirm = async (andUpload = false) => {
    try {
      await bulkSchedule(new Date(bulkStartAt).toISOString(), bulkIntervalDays * 1440);
      setReviewOpen(false);
      setFormOpen(false);
      setBulkStartAt('');
      if (andUpload) {
        setUploadNowLoading(true);
        try { await uploadNow(); } catch { /* logged in store */ }
        finally { setUploadNowLoading(false); }
      }
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
            {channelFiltered.length} item{channelFiltered.length !== 1 ? 's' : ''}
          </p>
        </div>
        {statusFilter === 'queue' && (
          <Button variant="outline" size="sm" onClick={() => setFormOpen((v) => !v)}>
            Bulk Schedule
          </Button>
        )}
      </div>

      {/* Status filter tabs */}
      <div className="flex items-center gap-1 rounded-lg bg-[#1A1A26] border border-border p-1 w-fit">
        {(
          [
            { key: 'queue' as const, label: 'Para Subir', count: queueCount },
            { key: 'history' as const, label: 'Histórico', count: historyCount },
          ] as const
        ).map(({ key, label, count }) => (
          <button
            key={key}
            onClick={() => handleStatusFilterChange(key)}
            className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
              statusFilter === key
                ? 'bg-primary text-primary-foreground'
                : 'text-zinc-400 hover:text-zinc-300'
            }`}
          >
            {label}
            {count > 0 && (
              <span
                className={`ml-1.5 rounded-full px-1.5 py-0.5 text-[10px] ${
                  statusFilter === key ? 'bg-white/20' : 'bg-zinc-700 text-zinc-400'
                }`}
              >
                {count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Review modal (bulk) */}
      <ReviewScheduleModal
        open={reviewOpen}
        mode="bulk"
        items={items}
        startAt={bulkStartAt ? new Date(bulkStartAt).toISOString() : undefined}
        intervalMinutes={bulkIntervalDays * 1440}
        channels={channelOptions}
        goUrl={goUrl}
        onConfirm={() => handleBulkConfirm(false)}
        onCancel={() => setReviewOpen(false)}
      />

      {/* Bulk schedule / upload-now form */}
      {formOpen && statusFilter === 'queue' && (
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
            <label className="text-xs text-zinc-400">Interval (days)</label>
            <input
              type="number"
              min={1}
              value={bulkIntervalDays}
              onChange={(e) => setBulkIntervalDays(Number(e.target.value))}
              className="w-24 rounded-lg bg-[#1A1A26] border border-border px-2.5 py-1.5 text-xs text-[#F0F0F8] focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
          <Button
            size="sm"
            disabled={!bulkStartAt || uploadNowLoading}
            onClick={() => void handleBulkConfirm(true)}
          >
            {uploadNowLoading ? 'Uploading…' : 'Schedule & Upload Now'}
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={!bulkStartAt}
            onClick={() => setReviewOpen(true)}
          >
            Preview & Schedule
          </Button>
          <Button size="sm" variant="ghost" onClick={() => setFormOpen(false)}>
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
      {!loading && channelFiltered.length === 0 && (
        <div className="rounded-xl border border-dashed border-zinc-700 p-10 text-center">
          <p className="text-sm text-zinc-500">
            {statusFilter === 'queue' ? 'Nenhum item na fila.' : 'Nenhum vídeo enviado ainda.'}
          </p>
          {statusFilter === 'queue' && (
            <p className="text-xs text-zinc-600 mt-1">
              Confirme um upload no pipeline para adicionar clips aqui.
            </p>
          )}
        </div>
      )}

      {/* Items — grouped by channel when "Todos" is selected and multiple channels exist */}
      <div className="space-y-6">
        {groups.map((group) => (
          <div key={group.channelId}>
            {selectedChannelId === null && channelOptions.length > 1 && (
              <h2 className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-3 pb-1 border-b border-zinc-800">
                {group.channelName}
              </h2>
            )}
            <div className="space-y-3">
              {group.items.map((item) => {
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
        ))}
      </div>
    </div>
  );
}
