'use client';

import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import type { Channel } from '@/store/channelStore';
import { useOAuthStore } from '@/store/oauthStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('ChannelAuthSection');

type AuthStatus = 'authorized' | 'expired' | 'not-authorized';

function getAuthStatus(channel: Channel): AuthStatus {
  const ch = channel as Channel & { AccessToken?: string; ExpiresAt?: number };
  if (!ch.AccessToken) return 'not-authorized';
  const nowMs = Date.now();
  // ExpiresAt from Go is stored as unix milliseconds
  if (ch.ExpiresAt && ch.ExpiresAt > 0 && ch.ExpiresAt <= nowMs) return 'expired';
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

interface AuthDialogState {
  channelId: number;
  channelName: string;
  authUrl: string;
}

export function ChannelAuthSection() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { channels, loading, fetchChannels } = useChannelStore();
  const { initOAuth, submitCode } = useOAuthStore();

  const [authorizing, setAuthorizing] = useState<number | null>(null);
  const [dialog, setDialog] = useState<AuthDialogState | null>(null);
  const [code, setCode] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [authError, setAuthError] = useState<{ channelId: number; message: string } | null>(null);

  useEffect(() => {
    void fetchChannels(goUrl);
  }, [goUrl, fetchChannels]);

  const handleAuthorize = async (channel: Channel) => {
    setAuthorizing(channel.ID);
    setAuthError(null);
    try {
      const result = await initOAuth(goUrl, channel.ID);
      setDialog({ channelId: channel.ID, channelName: channel.Name, authUrl: result.auth_url });
      setCode('');
      setSubmitError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to initiate OAuth';
      log.error('initOAuth failed', { channelId: channel.ID, err: message });
      setAuthError({ channelId: channel.ID, message });
    } finally {
      setAuthorizing(null);
    }
  };

  const handleSubmitCode = async () => {
    if (!dialog || !code.trim()) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      await submitCode(goUrl, dialog.channelId, code.trim());
      log.info('oauth code submitted successfully', { channelId: dialog.channelId });
      setDialog(null);
      setCode('');
      await fetchChannels(goUrl);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to submit code';
      log.error('submitCode failed', { channelId: dialog?.channelId, err: message });
      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
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
                        {ch.YouTubeChannelID || '—'}
                      </p>
                    </div>
                    <AuthBadge status={status} />
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs shrink-0"
                      disabled={isThisAuthorizing}
                      onClick={() => void handleAuthorize(ch)}
                    >
                      {isThisAuthorizing ? 'Loading…' : 'Authorize'}
                    </Button>
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

      <Dialog open={!!dialog} onOpenChange={(open) => { if (!open) setDialog(null); }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              Authorize {dialog?.channelName ?? 'Channel'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                Open the link below in your browser, sign in with Google, and copy the
                authorization code.
              </p>
              <div className="flex items-start gap-2">
                <textarea
                  readOnly
                  value={dialog?.authUrl ?? ''}
                  rows={3}
                  className="flex-1 resize-none rounded-md border border-input bg-muted px-3 py-2 text-xs font-mono text-foreground focus:outline-none"
                />
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 text-xs shrink-0 mt-0.5"
                  onClick={() => {
                    if (dialog?.authUrl) window.open(dialog.authUrl, '_blank');
                  }}
                >
                  Open
                </Button>
              </div>
            </div>

            <div className="space-y-1.5">
              <p className="text-sm font-medium text-foreground">Authorization code</p>
              <Input
                placeholder="Paste the code here"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                className="h-8 text-sm font-mono"
              />
              {submitError && (
                <p className="text-xs text-destructive">{submitError}</p>
              )}
            </div>
          </div>

          <DialogFooter showCloseButton>
            <Button
              size="sm"
              className="text-xs"
              disabled={submitting || !code.trim()}
              onClick={() => void handleSubmitCode()}
            >
              {submitting ? 'Confirming…' : 'Confirm'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
