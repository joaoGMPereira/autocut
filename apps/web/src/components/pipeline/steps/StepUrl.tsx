'use client';

import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { InfoBanner } from '@/components/ui/info-banner';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { useUrlHistoryStore } from '@/store/urlHistoryStore';
import { useChannelStore } from '@/store/channelStore';
import { ChannelCard, isAuthenticated } from '@/components/channels/ChannelCard';
import type { GatePayload, UrlHistoryEntry } from '@/types/pipeline';

function UrlHistorySection({ goUrl, onSelect }: { goUrl: string; onSelect: (url: string) => void }) {
  const history = useUrlHistoryStore((s) => s.history);
  const removeEntry = useUrlHistoryStore((s) => s.removeEntry);
  const clearAll = useUrlHistoryStore((s) => s.clearAll);

  if (history.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-subtle">Recent URLs</span>
        <button
          onClick={() => void clearAll(goUrl)}
          className="text-xs text-caption hover:text-prose transition-colors"
        >
          Clear all
        </button>
      </div>
      <ul className="space-y-1">
        {history.map((entry: UrlHistoryEntry) => (
          <li key={entry.id} className="flex items-center gap-2 group">
            <button
              onClick={() => onSelect(entry.url)}
              className="flex-1 min-w-0 text-left rounded px-2 py-1.5 hover:bg-surface transition-colors"
            >
              {entry.video_title && (
                <p className="text-xs font-medium text-foreground truncate">{entry.video_title}</p>
              )}
              <p className="text-xs text-subtle truncate">{entry.url}</p>
            </button>
            <button
              onClick={() => void removeEntry(goUrl, entry.id)}
              className="shrink-0 text-caption hover:text-prose transition-colors opacity-0 group-hover:opacity-100 px-1"
              aria-label="Remove"
            >
              ×
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

interface StepUrlProps {
  historical?: GatePayload;
}

export function StepUrl({ historical }: StepUrlProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const createRun = usePipelineStore((s) => s.createRun);
  const advance = usePipelineStore((s) => s.advance);
  const subscribeSSE = usePipelineStore((s) => s.subscribeSSE);
  const resubmitFromBacked = usePipelineStore((s) => s.resubmitFromBacked);
  const isLoading = usePipelineStore((s) => s.isLoading);
  const error = usePipelineStore((s) => s.error);
  const videoReused = usePipelineStore((s) => s.videoReused);
  const pendingReuse = usePipelineStore((s) => s.pendingReuse);
  const confirmReuse = usePipelineStore((s) => s.confirmReuse);
  const redownload = usePipelineStore((s) => s.redownload);
  const fetchHistory = useUrlHistoryStore((s) => s.fetchHistory);
  const channels = useChannelStore((s) => s.channels);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  const historicalUrl = historical?.kind === 'url' ? historical.url : undefined;
  const [url, setUrl] = useState(historicalUrl ?? '');

  // Select favorite authenticated channel by default
  const defaultChannel = channels.find((c) => isAuthenticated(c) && c.IsFavorite) ?? channels.find(isAuthenticated) ?? null;
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);

  useEffect(() => {
    void fetchHistory(goUrl);
    if (channels.length === 0) void fetchChannels(goUrl);
  }, [goUrl, fetchHistory]); // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-select favorite once channels load
  useEffect(() => {
    if (selectedChannelId === null && defaultChannel) {
      setSelectedChannelId(defaultChannel.ID);
    }
  }, [channels]); // eslint-disable-line react-hooks/exhaustive-deps

  const isHistorical = historical !== undefined;
  const selectedChannel = channels.find((c) => c.ID === selectedChannelId) ?? null;
  const canStart = !!selectedChannelId && !!selectedChannel && isAuthenticated(selectedChannel);

  const handleStart = async () => {
    const trimmed = url.trim();
    if (!trimmed || !selectedChannelId) return;
    const runId = await createRun(goUrl);
    if (!runId) return;
    // advance BEFORE subscribeSSE: the backend publishes state_changed SSE
    // synchronously during Advance for the reuse case. If SSE is connected
    // first, the event arrives before the HTTP response sets pendingReuse,
    // causing auto-navigation that skips the reuse card.
    await advance(goUrl, runId, { url: trimmed, channel_id: selectedChannelId });
    subscribeSSE(goUrl, runId);
  };

  const handleResubmit = async () => {
    const trimmed = url.trim();
    if (!trimmed) return;
    await resubmitFromBacked(goUrl, 'WAITING_URL', { kind: 'url', url: trimmed });
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !isLoading && canStart) {
      if (isHistorical) void handleResubmit();
      else void handleStart();
    }
  };

  return (
    <div className="space-y-6 max-w-lg">
      <div className="space-y-1">
        <h2 className="font-display text-2xl font-bold tracking-[-0.02em] text-heading">Video URL</h2>
        <p className="text-sm text-subtle">Paste a YouTube or Twitch URL to start the pipeline.</p>
      </div>

      {isHistorical && (
        <InfoBanner>
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </InfoBanner>
      )}

      {/* Channel selection */}
      {!isHistorical && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-subtle">Canal</p>
          {channels.length === 0 ? (
            <p className="text-xs text-caption">Nenhum canal configurado. Adicione um canal nas configurações.</p>
          ) : (
            <div className="flex flex-col gap-2">
              {channels.map((ch) => (
                <ChannelCard
                  key={ch.ID}
                  channel={ch}
                  goUrl={goUrl}
                  selected={selectedChannelId === ch.ID}
                  onSelect={(c) => setSelectedChannelId(c.ID)}
                  showOAuth={!isAuthenticated(ch)}
                />
              ))}
            </div>
          )}
        </div>
      )}

      <div className="space-y-3">
        <Input
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="https://www.youtube.com/watch?v=..."
          disabled={isLoading}
          data-testid="step-url-input"
        />

        <Button
          variant="brand"
          onClick={() => void (isHistorical ? handleResubmit() : handleStart())}
          disabled={isLoading || !url.trim() || (!isHistorical && !canStart)}
          data-testid="step-url-submit"
          className="w-full"
        >
          {isLoading ? 'Starting…' : isHistorical ? 'Re-submit' : 'Start Pipeline'}
        </Button>

        {!isHistorical && !canStart && channels.length > 0 && (
          <p className="text-xs text-subtle text-center">
            {!selectedChannelId ? 'Selecione um canal para continuar' : 'Autorize o canal para continuar'}
          </p>
        )}

        {error && (
          <p className="text-xs text-destructive">{error}</p>
        )}

        {!isHistorical && <UrlHistorySection goUrl={goUrl} onSelect={(u) => setUrl(u)} />}
      </div>

      {pendingReuse && videoReused && (
        <div className="space-y-3 rounded-md border border-border bg-card p-4">
          <p className="text-sm text-prose">Video já baixado anteriormente</p>
          <div className="flex gap-2">
            <Button
              variant="brand"
              onClick={() => confirmReuse()}
              className="flex-1"
            >
              Reusar e continuar
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                const runId = usePipelineStore.getState().activeRunId;
                if (runId) {
                  void redownload(goUrl, runId);
                }
                confirmReuse();
              }}
            >
              Re-baixar
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
