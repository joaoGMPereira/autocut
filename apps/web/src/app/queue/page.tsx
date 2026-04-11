'use client';

import { useEffect, useState } from 'react';
import { Anton } from 'next/font/google';
import { RefreshCw, Save, CalendarRange } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useQueueStore } from '@/store/queueStore';
import { QueueItemCard } from '@/components/queue/QueueItemCard';
import { BulkScheduleModal } from '@/components/queue/BulkScheduleModal';
import { createLogger } from '@/lib/logger';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });
const log = createLogger('QueuePage');

export default function QueuePage() {
  const { items, loading, error, fetchQueue, saveLocal, bulkSchedule } = useQueueStore();
  const [bulkModalOpen, setBulkModalOpen] = useState(false);
  const [saveLoading, setSaveLoading] = useState(false);

  useEffect(() => {
    void fetchQueue();
  }, [fetchQueue]);

  const handleSave = async () => {
    setSaveLoading(true);
    try {
      await saveLocal();
    } catch (err) {
      log.error('save local failed', { err });
    } finally {
      setSaveLoading(false);
    }
  };

  const handleBulkSchedule = async (startAt: string, intervalMinutes: number) => {
    await bulkSchedule(startAt, intervalMinutes);
  };

  const queueableItems = items.filter((item) => item.status === 'queued' || item.status === 'failed');

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
          UPLOAD QUEUE
        </h1>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            className="h-8 gap-1.5 text-xs"
            disabled={loading}
            onClick={() => void fetchQueue()}
          >
            <RefreshCw className={`size-3.5 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>

          <Button
            variant="outline"
            size="sm"
            className="h-8 gap-1.5 text-xs"
            disabled={saveLoading}
            onClick={() => void handleSave()}
          >
            <Save className="size-3.5" />
            {saveLoading ? 'Saving…' : 'Save Queue'}
          </Button>

          <Button
            size="sm"
            className="h-8 gap-1.5 text-xs"
            disabled={queueableItems.length === 0}
            onClick={() => setBulkModalOpen(true)}
          >
            <CalendarRange className="size-3.5" />
            Bulk Schedule
          </Button>
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="rounded-xl bg-destructive/10 border border-destructive/20 px-4 py-3">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      {/* Items list */}
      {items.length === 0 ? (
        <div className="rounded-xl bg-card border border-border p-5">
          <p className="text-[#5C5C80] text-sm text-center py-8">
            {loading ? 'Loading queue…' : 'No items in queue'}
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {items.map((item) => (
            <QueueItemCard key={item.id} item={item} />
          ))}
        </div>
      )}

      {/* Bulk schedule modal */}
      <BulkScheduleModal
        open={bulkModalOpen}
        onOpenChange={setBulkModalOpen}
        itemCount={queueableItems.length}
        onConfirm={handleBulkSchedule}
      />
    </div>
  );
}
