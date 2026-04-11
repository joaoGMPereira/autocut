'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import type { Channel } from '@/store/channelStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('ChannelsSection');

type OAuthStatus = 'authorized' | 'expired' | 'not-connected';

function getOAuthStatus(channel: Channel): OAuthStatus {
  // Channel from Go: AccessToken is the token field, ExpiresAt is unix seconds
  const ch = channel as Channel & { AccessToken?: string; ExpiresAt?: number };
  if (!ch.AccessToken) return 'not-connected';
  const nowSec = Math.floor(Date.now() / 1000);
  if (ch.ExpiresAt && ch.ExpiresAt > 0 && ch.ExpiresAt <= nowSec) return 'expired';
  return 'authorized';
}

function OAuthBadge({ status }: { status: OAuthStatus }) {
  if (status === 'authorized') {
    return (
      <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-emerald-400/20 text-emerald-400">
        Authorized
      </span>
    );
  }
  if (status === 'expired') {
    return (
      <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-amber-400/20 text-amber-400">
        Token Expired
      </span>
    );
  }
  return (
    <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-zinc-600/40 text-zinc-400">
      Not Connected
    </span>
  );
}

export function ChannelsSection() {
  const router = useRouter();
  const goUrl = useAppStore((s) => s.goUrl);
  const { channels, loading, error, fetchChannels, removeChannel } = useChannelStore();

  useEffect(() => {
    void fetchChannels(goUrl);
  }, [goUrl, fetchChannels]);

  const handleRevoke = async (id: number, name: string) => {
    log.info('revoking channel', { id, name });
    try {
      await removeChannel(goUrl, id);
    } catch (err) {
      log.error('revoke failed', { id, err });
    }
  };

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">OAuth / Channels</h2>
        <Button
          size="sm"
          className="h-7 text-xs"
          onClick={() => router.push('/channels')}
        >
          + Add Channel
        </Button>
      </div>

      {error && (
        <div className="rounded-xl bg-destructive/10 border border-destructive/20 px-4 py-3">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      <div className="rounded-xl border border-border bg-card">
        {loading && channels.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            Loading channels…
          </div>
        ) : channels.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            No channels configured.{' '}
            <button
              className="text-primary underline underline-offset-2"
              onClick={() => router.push('/channels')}
            >
              Add one
            </button>
          </div>
        ) : (
          channels.map((ch, i) => {
            const status = getOAuthStatus(ch);
            return (
              <div key={ch.ID}>
                {i > 0 && <Separator />}
                <div className="flex items-center gap-3 px-5 py-3.5">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-foreground truncate">{ch.Name}</p>
                    <p className="text-[11px] font-mono text-muted-foreground truncate">
                      {ch.YouTubeChannelID}
                    </p>
                  </div>
                  <OAuthBadge status={status} />
                  <Button
                    variant="destructive"
                    size="sm"
                    className="h-7 text-xs shrink-0"
                    onClick={() => void handleRevoke(ch.ID, ch.Name)}
                  >
                    Revoke
                  </Button>
                </div>
              </div>
            );
          })
        )}
      </div>
    </section>
  );
}
