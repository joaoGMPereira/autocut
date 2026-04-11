'use client';

import { useState, useRef, useEffect } from 'react';
import { VideoInfoCard } from './VideoInfoCard';
import type { VideoInfoData, DownloadSSEEvent } from '@/types/download';

const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

type Status = 'idle' | 'extracting' | 'downloading' | 'done' | 'error';

export function DownloadTestClient() {
  const [url, setUrl] = useState('');
  const [status, setStatus] = useState<Status>('idle');
  const [videoInfo, setVideoInfo] = useState<VideoInfoData | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    return () => {
      esRef.current?.close();
    };
  }, []);

  async function handleStart() {
    if (!url.trim()) return;

    esRef.current?.close();
    setVideoInfo(null);
    setErrorMsg(null);
    setStatus('extracting');

    let jobId: string;
    try {
      const res = await fetch(`${goUrl}/api/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: url.trim(), type: 'youtube', output_dir: '/tmp/autocut-test' }),
      });
      if (!res.ok) throw new Error(`Server error: ${res.status}`);
      const data = (await res.json()) as { job_id: string };
      jobId = data.job_id;
    } catch (err) {
      setStatus('error');
      setErrorMsg(err instanceof Error ? err.message : 'Failed to start download');
      return;
    }

    const es = new EventSource(`${goUrl}/api/download/${jobId}/stream`);
    esRef.current = es;

    es.onmessage = (e: MessageEvent) => {
      try {
        const evt = JSON.parse(e.data as string) as DownloadSSEEvent;

        if (evt.type === 'video_info') {
          setVideoInfo({
            title: evt.data.title,
            thumbnailUrl: evt.data.thumbnail_url,
            durationSec: evt.data.duration_sec,
          });
          setStatus('downloading');
        }

        if (evt.type === 'done') {
          setStatus('done');
          es.close();
        }

        if (evt.type === 'error') {
          setStatus('error');
          setErrorMsg(evt.data.message);
          es.close();
        }
      } catch {
        // ignore parse errors
      }
    };

    es.onerror = () => {
      setStatus('error');
      setErrorMsg('SSE connection lost');
      es.close();
    };
  }

  const statusLabel: Record<Status, string> = {
    idle: '',
    extracting: 'Extracting metadata…',
    downloading: 'Downloading…',
    done: 'Done',
    error: '',
  };

  return (
    <div className="space-y-6 max-w-md">
      <div className="space-y-3">
        <input
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://www.youtube.com/watch?v=..."
          className="w-full bg-zinc-800 border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder-zinc-500 focus:outline-none focus:border-cyan-500"
        />
        <button
          type="button"
          onClick={handleStart}
          disabled={!url.trim() || status === 'extracting' || status === 'downloading'}
          className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed text-sm font-medium text-white transition-colors"
        >
          {status === 'extracting' || status === 'downloading' ? 'Running…' : 'Start Download'}
        </button>
      </div>

      {statusLabel[status] && (
        <p className="text-sm text-zinc-400">{statusLabel[status]}</p>
      )}

      {errorMsg && (
        <p className="text-sm text-red-400">{errorMsg}</p>
      )}

      {videoInfo && <VideoInfoCard info={videoInfo} />}
    </div>
  );
}
