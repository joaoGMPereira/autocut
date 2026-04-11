'use client';

import { useState, useEffect } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { GateActionBar } from '../GateActionBar';
import { useChannelStore } from '@/store/channelStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('StepUrl');
const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

export function StepUrl() {
  const run = usePipelineStore((s) => s.run);
  const advance = usePipelineStore((s) => s.advance);
  const createRun = usePipelineStore((s) => s.createRun);
  const subscribeSSE = usePipelineStore((s) => s.subscribeSSE);

  const channels = useChannelStore((s) => s.channels);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  useEffect(() => {
    fetchChannels(goUrl);
  }, [fetchChannels, goUrl]);

  const [url, setUrl] = useState('');
  const [channelId, setChannelId] = useState<number | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // Auto-preview YouTube thumbnail
  function handleUrlChange(v: string) {
    setUrl(v);
    const match = v.match(/[?&]v=([^&]+)/);
    if (match) {
      setPreview(`https://img.youtube.com/vi/${match[1]}/hqdefault.jpg`);
    } else {
      setPreview(null);
    }
  }

  async function handleNext() {
    if (!url.trim()) return;
    setLoading(true);
    try {
      let runId = run?.id ?? null;
      if (!runId) {
        runId = await createRun(goUrl);
        if (!runId) return;
        subscribeSSE(goUrl, runId);
      }
      await advance(goUrl, runId, { url: url.trim(), channel_id: channelId ?? undefined });
    } catch (err) {
      log.error('[StepUrl] advance failed', { error: err instanceof Error ? err.message : 'Unknown' });
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6 max-w-lg">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">Video URL</h2>
        <p className="text-sm text-zinc-400">Paste a YouTube or Twitch URL to begin.</p>
      </div>

      <div className="space-y-3">
        <input
          type="url"
          value={url}
          onChange={(e) => handleUrlChange(e.target.value)}
          placeholder="https://www.youtube.com/watch?v=..."
          className="w-full bg-zinc-800 border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder-zinc-500 focus:outline-none focus:border-cyan-500"
        />

        <select
          value={channelId ?? ''}
          onChange={(e) => setChannelId(e.target.value ? Number(e.target.value) : null)}
          className="w-full bg-zinc-800 border border-border rounded-lg px-3 py-2 text-sm text-foreground focus:outline-none focus:border-cyan-500"
        >
          <option value="">— No channel —</option>
          {channels.map((ch) => (
            <option key={ch.ID} value={ch.ID}>{ch.Name}</option>
          ))}
        </select>

        {preview && (
          <div className="w-full aspect-video rounded-lg overflow-hidden bg-zinc-800">
            <img src={preview} alt="Video preview" className="w-full h-full object-cover" />
          </div>
        )}
      </div>

      <GateActionBar>
        <button
          type="button"
          onClick={handleNext}
          disabled={!url.trim() || loading}
          className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed text-sm font-medium text-white transition-colors"
        >
          {loading ? 'Starting…' : 'Next →'}
        </button>
      </GateActionBar>
    </div>
  );
}
