'use client';

import { useRef, useState } from 'react';
import { useChannelStore } from '@/store/channelStore';
import type { Channel } from '@/store/channelStore';
import { useOAuthStore } from '@/store/oauthStore';
import { createLogger } from '@/lib/logger';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

const log = createLogger('ChannelCard');

const POLL_INTERVAL_MS = 2000;
const POLL_MAX_ITERATIONS = 150; // 5 min

export type AuthStatus = 'authorized' | 'expired' | 'not-authorized';

export function getAuthStatus(ch: Channel): AuthStatus {
  if (!ch.AccessToken) return 'not-authorized';
  const nowMs = Date.now();
  if (ch.ExpiresAt > 0 && ch.ExpiresAt <= nowMs) return 'expired';
  return 'authorized';
}

export function isAuthenticated(ch: Channel): boolean {
  const s = getAuthStatus(ch);
  return s === 'authorized' || s === 'expired';
}

function AuthBadge({ status }: { status: AuthStatus }) {
  if (status === 'authorized') {
    return <Badge variant="success" className="shrink-0">Authorized</Badge>;
  }
  if (status === 'expired') {
    return <Badge variant="warning" className="shrink-0">Expirado</Badge>;
  }
  return <Badge variant="secondary" className="shrink-0">Não conectado</Badge>;
}

interface ChannelAvatarProps {
  channel: Channel;
  goUrl: string;
  size?: 'sm' | 'md';
}

export function ChannelAvatar({ channel, goUrl, size = 'md' }: ChannelAvatarProps) {
  const dim = size === 'sm' ? 'h-8 w-8 text-xs' : 'h-10 w-10 text-sm';
  const initial = (channel.ChannelTitle || channel.Name).charAt(0).toUpperCase();
  const bg = `hsl(${(channel.ID * 67) % 360}, 55%, 40%)`;
  return (
    <div className={`relative ${dim} shrink-0`}>
      <img
        src={`${goUrl}/api/channels/${channel.ID}/avatar`}
        alt=""
        className={`${dim} rounded-full object-cover`}
        onError={(e) => {
          (e.currentTarget as HTMLImageElement).style.display = 'none';
          const sib = e.currentTarget.nextElementSibling as HTMLElement | null;
          if (sib) sib.style.display = 'flex';
        }}
      />
      <div
        className={`${dim} rounded-full items-center justify-center font-bold text-white`}
        style={{ display: 'none', background: bg }}
      >
        {initial}
      </div>
    </div>
  );
}

interface ChannelCardProps {
  channel: Channel;
  goUrl: string;
  /** Highlight as selected (picker mode) */
  selected?: boolean;
  /** Called when card is clicked; only fires for authenticated channels */
  onSelect?: (channel: Channel) => void;
  /** Show Authorize / Cancel button */
  showOAuth?: boolean;
  /** Slot for extra trailing actions (e.g. Revoke button) */
  actions?: React.ReactNode;
}

export function ChannelCard({
  channel,
  goUrl,
  selected,
  onSelect,
  showOAuth = false,
  actions,
}: ChannelCardProps) {
  const { fetchChannels } = useChannelStore();
  const { initOAuth } = useOAuthStore();

  const [authorizing, setAuthorizing] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollCountRef = useRef(0);

  const status = getAuthStatus(channel);
  // authorized + expired are selectable (backend refreshes expired tokens automatically)
  const canSelect = status === 'authorized' || status === 'expired';

  const stopPolling = () => {
    if (pollingRef.current) clearInterval(pollingRef.current);
    pollingRef.current = null;
    pollCountRef.current = 0;
  };

  const handleAuthorize = async () => {
    setAuthorizing(true);
    setAuthError(null);
    try {
      const result = await initOAuth(goUrl, channel.ID);
      window.open(result.auth_url, '_blank');

      pollingRef.current = setInterval(() => {
        pollCountRef.current += 1;
        if (pollCountRef.current >= POLL_MAX_ITERATIONS) {
          stopPolling();
          setAuthorizing(false);
          setAuthError('Timeout — tente novamente');
          return;
        }
        void (async () => {
          try {
            await fetchChannels(goUrl);
            const updated = useChannelStore.getState().channels.find((c) => c.ID === channel.ID);
            if (updated?.AccessToken) {
              stopPolling();
              setAuthorizing(false);
              log.info('oauth authorized', { channelId: channel.ID });
            }
          } catch { /* ignore */ }
        })();
      }, POLL_INTERVAL_MS);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Falha ao iniciar OAuth';
      log.error('initOAuth failed', { channelId: channel.ID, err: msg });
      setAuthError(msg);
      setAuthorizing(false);
    }
  };

  const handleCancel = () => {
    stopPolling();
    setAuthorizing(false);
    log.info('oauth cancelled', { channelId: channel.ID });
  };

  const isPicker = onSelect !== undefined;

  return (
    <div
      onClick={() => canSelect && onSelect?.(channel)}
      className={[
        'flex flex-col gap-1.5 rounded-lg border px-3 py-2.5 transition-all',
        isPicker ? (canSelect ? 'cursor-pointer' : 'cursor-not-allowed opacity-50') : '',
        isPicker && selected
          ? 'border-brand/60 bg-brand/5 ring-1 ring-brand/30'
          : 'border-border bg-background',
        isPicker && canSelect && !selected ? 'hover:border-border/70 hover:bg-surface/50' : '',
      ].join(' ')}
    >
      <div className="flex items-center gap-3">
        <ChannelAvatar channel={channel} goUrl={goUrl} size={isPicker ? 'sm' : 'md'} />

        <div className="flex-1 min-w-0">
          <p className="text-xs font-semibold text-heading truncate">
            {channel.ChannelTitle || channel.Name}
          </p>
          {channel.ChannelID && (
            <p className="text-[10px] font-mono text-subtle truncate">{channel.ChannelID}</p>
          )}
        </div>

        <AuthBadge status={status} />

        {showOAuth && (
          authorizing ? (
            <>
              <span className="text-[10px] text-subtle shrink-0">Aguardando…</span>
              <Button
                variant="outline"
                size="sm"
                className="h-6 text-[10px] shrink-0"
                onClick={(e) => { e.stopPropagation(); handleCancel(); }}
              >
                Cancelar
              </Button>
            </>
          ) : (
            <Button
              variant="outline"
              size="sm"
              className="h-6 text-[10px] shrink-0"
              onClick={(e) => { e.stopPropagation(); void handleAuthorize(); }}
            >
              {status === 'not-authorized' ? 'Autorizar' : 'Re-autorizar'}
            </Button>
          )
        )}

        {actions}

        {isPicker && selected && (
          <span className="flex h-4 w-4 items-center justify-center rounded-full bg-brand text-brand-foreground text-[10px] font-bold shrink-0">✓</span>
        )}
      </div>

      {authError && (
        <p className="text-[10px] text-destructive pl-1">{authError}</p>
      )}
    </div>
  );
}
