'use client';

import { useEffect } from 'react';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import { ChannelCard } from '@/components/channels/ChannelCard';

export function ChannelAuthSection() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { channels, loading, fetchChannels } = useChannelStore();

  useEffect(() => {
    void fetchChannels(goUrl);
  }, [goUrl, fetchChannels]);

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-foreground">Channel Authorization</h2>

      <div className="rounded-xl border border-border bg-card divide-y divide-border">
        {loading && channels.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            Loading channels…
          </div>
        ) : channels.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            No channels configured.
          </div>
        ) : (
          channels.map((ch, i) => (
            <div key={ch.ID}>
              {i > 0 && <Separator />}
              <div className="px-4 py-3">
                <ChannelCard channel={ch} goUrl={goUrl} showOAuth />
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
