'use client';

import { useEffect, useRef } from 'react';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import type { Channel } from '@/store/channelStore';
import { useOAuthStore } from '@/store/oauthStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('ChannelAuthSection');

const POLL_INTERVAL_MS = 2000;
const POLL_MAX_ITERATIONS = 150; // 5 minutes

type AuthStatus = 'authorized' | 'expired' | 'not-authorized';

function getAuthStatus(channel: Channel): AuthStatus {
  if (!channel.AccessToken) return 'not-authorized';
  const nowMs = Date.now();
  if (channel.ExpiresAt > 0 && channel.ExpiresAt <= nowMs) return 'expired';
  return 'authorized';
}

function AuthBadge({ status }: { status: AuthStatus }) {
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
      Not Authorized
    </span>
  );
}

export function ChannelAuthSection() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { channels, loading, fetchChannels } = useChannelStore();
  const { initOAuth } = useOAuthStore();

  const [authorizing, setAuthorizing] = useState<number | null>(null);
  const [authError, setAuthError] = useState<{ channelId: number; message: string } | null>(null);

  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollCountRef = useRef(0);

  useEffect(() => {
    void fetchChannels(goUrl);
  }, [goUrl, fetchChannels]);

  useEffect(() => {
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, []);

  const stopPolling = () => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
    pollCountRef.current = 0;
  };

  const handleAuthorize = async (channel: Channel) => {
    setAuthorizing(channel.ID);
    setAuthError(null);
    try {
      const result = await initOAuth(goUrl, channel.ID);
      window.open(result.auth_url, '_blank');

      pollCountRef.current = 0;
      pollingRef.current = setInterval(() => {
        pollCountRef.current += 1;

        if (pollCountRef.current >= POLL_MAX_ITERATIONS) {
          stopPolling();
          setAuthorizing(null);
          setAuthError({ channelId: channel.ID, message: 'Authorization timed out — try again' });
          return;
        }

        void (async () => {
          try {
            await fetchChannels(goUrl);
            const updated = useChannelStore.getState().channels.find((c) => c.ID === channel.ID);
            if (updated?.AccessToken) {
              stopPolling();
              setAuthorizing(null);
              log.info('oauth authorized', { channelId: channel.ID });
            }
          } catch {
            // ignore poll errors, keep polling
          }
        })();
      }, POLL_INTERVAL_MS);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to initiate OAuth';
      log.error('initOAuth failed', { channelId: channel.ID, err: message });
      setAuthError({ channelId: channel.ID, message });
      setAuthorizing(null);
    }
  };

  const handleCancel = (channelId: number) => {
    stopPolling();
    setAuthorizing(null);
    log.info('oauth authorization cancelled', { channelId });
  };

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-foreground">Channel Authorization</h2>

      <div className="rounded-xl border border-border bg-card">
        {loading && channels.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            Loading channels…
          </div>
        ) : channels.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted-foreground">
            No channels configured.
          </div>
        ) : (
          channels.map((ch, i) => {
            const status = getAuthStatus(ch);
            const isThisAuthorizing = authorizing === ch.ID;
            const error = authError?.channelId === ch.ID ? authError.message : null;
            return (
              <div key={ch.ID}>
                {i > 0 && <Separator />}
                <div className="flex flex-col px-5 py-3.5 gap-1.5">
                  <div className="flex items-center gap-3">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-foreground truncate">{ch.Name}</p>
                      <p className="text-[11px] font-mono text-muted-foreground truncate">
                        {ch.ChannelID || ch.YouTubeChannelID || '—'}
                      </p>
                    </div>
                    <AuthBadge status={status} />
                    {isThisAuthorizing ? (
                      <>
                        <span className="text-xs text-muted-foreground shrink-0">
                          Waiting…
                        </span>
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-7 text-xs shrink-0"
                          onClick={() => handleCancel(ch.ID)}
                        >
                          Cancel
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 text-xs shrink-0"
                        onClick={() => void handleAuthorize(ch)}
                      >
                        Authorize
                      </Button>
                    )}
                  </div>
                  {error && (
                    <p className="text-xs text-destructive pl-0.5">{error}</p>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>
    </section>
  );
}
