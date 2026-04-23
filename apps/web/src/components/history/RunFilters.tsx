'use client';

import { createLogger } from '@/lib/logger';
import { useHistoryStore } from '@/store/historyStore';
import { useChannelStore } from '@/store/channelStore';
import { useAppStore } from '@/store/appStore';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useEffect } from 'react';

const log = createLogger('RunFilters');

const STATUS_OPTIONS = [
  { value: 'all', label: 'All Statuses' },
  { value: 'done', label: 'Done' },
  { value: 'error', label: 'Error' },
  { value: 'running', label: 'Running' },
  { value: 'pending', label: 'Pending' },
];

export function RunFilters() {
  const goUrl = useAppStore((s) => s.goUrl);
  const filters = useHistoryStore((s) => s.filters);
  const setFilter = useHistoryStore((s) => s.setFilter);
  const resetFilters = useHistoryStore((s) => s.resetFilters);
  const fetchRuns = useHistoryStore((s) => s.fetchRuns);
  const channels = useChannelStore((s) => s.channels);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  useEffect(() => {
    if (goUrl && channels.length === 0) {
      log.info('fetching channels for filter dropdown');
      void fetchChannels(goUrl);
    }
  }, [goUrl, channels.length, fetchChannels]);

  const handleChannelChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value;
    const id = val === '' ? null : Number(val);
    log.info('filter channel changed', { channelId: id });
    setFilter('channelId', id);
    void fetchRuns(goUrl);
  };

  const handleStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    log.info('filter status changed', { status: e.target.value });
    setFilter('status', e.target.value);
    void fetchRuns(goUrl);
  };

  const handleDateFromChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFilter('dateFrom', e.target.value);
    void fetchRuns(goUrl);
  };

  const handleDateToChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFilter('dateTo', e.target.value);
    void fetchRuns(goUrl);
  };

  const handleReset = () => {
    log.info('resetting filters');
    resetFilters();
    void fetchRuns(goUrl);
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      {/* Channel selector */}
      <select
        value={filters.channelId ?? ''}
        onChange={handleChannelChange}
        className="h-9 rounded-md border border-border bg-background px-3 text-sm text-heading focus:outline-none focus:ring-1 focus:ring-brand/40"
        aria-label="Filter by channel"
      >
        <option value="">All Channels</option>
        {channels.map((ch) => (
          <option key={ch.ID} value={ch.ID}>
            {ch.Name}
          </option>
        ))}
      </select>

      {/* Status selector */}
      <select
        value={filters.status}
        onChange={handleStatusChange}
        className="h-9 rounded-md border border-border bg-background px-3 text-sm text-heading focus:outline-none focus:ring-1 focus:ring-brand/40"
        aria-label="Filter by status"
      >
        {STATUS_OPTIONS.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>

      {/* Date from */}
      <Input
        type="date"
        value={filters.dateFrom}
        onChange={handleDateFromChange}
        aria-label="From date"
        className="h-9 w-40 bg-background border-border text-sm text-heading"
      />

      {/* Date to */}
      <Input
        type="date"
        value={filters.dateTo}
        onChange={handleDateToChange}
        aria-label="To date"
        className="h-9 w-40 bg-background border-border text-sm text-heading"
      />

      {/* Reset */}
      <Button
        variant="ghost"
        size="sm"
        onClick={handleReset}
        className="text-subtle hover:text-heading"
      >
        Reset
      </Button>
    </div>
  );
}
