'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import { ChannelsSection } from '@/components/settings/ChannelsSection';
import { ChannelAuthSection } from '@/components/settings/ChannelAuthSection';
import { createLogger } from '@/lib/logger';

const log = createLogger('[ChannelsPage]');

function AddChannelForm() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { addChannel, fetchChannels } = useChannelStore();
  const [name, setName] = useState('');
  const [youtubeChannelId, setYoutubeChannelId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      await addChannel(goUrl, { name: name.trim(), youtube_channel_id: youtubeChannelId.trim() });
      await fetchChannels(goUrl);
      setName('');
      setYoutubeChannelId('');
      log.info('channel added', { name });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to add channel';
      log.error('add channel failed', { err: message });
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={(e) => void handleSubmit(e)} className="rounded-xl border border-border bg-card p-5 space-y-4">
      <h2 className="text-base font-semibold text-foreground">Add Channel</h2>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex-1 space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground" htmlFor="channel-name">
            Name <span className="text-destructive">*</span>
          </label>
          <Input
            id="channel-name"
            placeholder="e.g. cortes_inerd"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="h-8 text-sm"
            required
          />
        </div>
        <div className="flex-1 space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground" htmlFor="yt-channel-id">
            YouTube Channel ID
          </label>
          <Input
            id="yt-channel-id"
            placeholder="UCxxxxxxxxxxxxxxxxxx"
            value={youtubeChannelId}
            onChange={(e) => setYoutubeChannelId(e.target.value)}
            className="h-8 text-sm font-mono"
          />
        </div>
        <Button type="submit" size="sm" className="h-8 text-xs shrink-0" disabled={submitting || !name.trim()}>
          {submitting ? 'Adding…' : 'Add Channel'}
        </Button>
      </div>
      {error && <p className="text-xs text-destructive">{error}</p>}
    </form>
  );
}

export default function ChannelsPage() {
  return (
    <div className="flex flex-col gap-6 p-6 max-w-3xl mx-auto w-full">
      <h1 className="font-display text-[28px] font-bold tracking-[-0.02em] text-heading">Channels</h1>
      <AddChannelForm />
      <ChannelsSection />
      <ChannelAuthSection />
    </div>
  );
}
