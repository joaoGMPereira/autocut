'use client';

import { useState } from 'react';
import { Separator } from '@/components/ui/separator';
import { Button } from '@/components/ui/button';
import { HighlightStrategySelector } from '@/components/highlights/HighlightStrategySelector';
import { ConfidenceThresholdSlider } from '@/components/highlights/ConfidenceThresholdSlider';
import { HighlightList } from '@/components/highlights/HighlightList';
import { useHighlightStore } from '@/store/highlightStore';
import { useAppStore } from '@/store/appStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('HighlightsClient');

export function HighlightsClient() {
  const { goUrl } = useAppStore();
  const {
    highlights,
    strategies,
    threshold,
    chatJsonPath,
    status,
    toggleStrategy,
    setThreshold,
    setChatJsonPath,
    runDetection,
    reset,
  } = useHighlightStore();

  const [videoPath, setVideoPath] = useState('');
  const [error, setError] = useState<string | null>(null);

  const isRunning = status === 'running';

  const handleRun = async () => {
    if (!videoPath.trim()) {
      setError('Informe o caminho do vídeo.');
      return;
    }
    if (strategies.length === 0) {
      setError('Selecione ao menos uma strategy.');
      return;
    }
    setError(null);
    log.info('starting detection', { videoPath, strategies, threshold });
    try {
      await runDetection(videoPath.trim(), goUrl);
    } catch (err) {
      log.error('runDetection threw', { err });
      setError(err instanceof Error ? err.message : 'Erro desconhecido');
    }
  };

  const handleReset = () => {
    reset();
    setError(null);
  };

  return (
    <div className="flex flex-col gap-6 lg:flex-row lg:gap-8 lg:items-start">
      {/* Left panel: form */}
      <div className="flex flex-col gap-5 lg:w-[380px] shrink-0">
        <div className="rounded-xl border border-zinc-800 bg-card p-5 flex flex-col gap-5">
          {/* Video path */}
          <label className="flex flex-col gap-1.5">
            <span className="text-xs font-medium text-zinc-400">Caminho do Vídeo</span>
            <input
              type="text"
              value={videoPath}
              onChange={(e) => setVideoPath(e.target.value)}
              placeholder="/path/to/video.mp4"
              disabled={isRunning}
              className="w-full rounded-lg bg-zinc-900 border border-zinc-700 px-3 py-2 text-sm focus:outline-none focus:border-blue-500/60 placeholder:text-zinc-600 text-zinc-200 disabled:opacity-50"
            />
          </label>

          <Separator className="bg-zinc-800" />

          {/* Strategy selector */}
          <HighlightStrategySelector
            selected={strategies}
            chatJsonPath={chatJsonPath}
            onToggle={toggleStrategy}
            onChatJsonPathChange={setChatJsonPath}
          />

          <Separator className="bg-zinc-800" />

          {/* Threshold slider */}
          <ConfidenceThresholdSlider value={threshold} onChange={setThreshold} />

          {/* Error */}
          {error && (
            <p className="text-xs text-red-400">{error}</p>
          )}

          {/* Actions */}
          <div className="flex gap-2">
            <Button
              onClick={handleRun}
              disabled={isRunning || !videoPath.trim() || strategies.length === 0}
              className="flex-1 bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50"
            >
              {isRunning ? 'Detectando…' : 'Detect Highlights'}
            </Button>
            {(status === 'done' || status === 'error') && (
              <Button
                variant="outline"
                onClick={handleReset}
                className="border-zinc-700 text-zinc-400 hover:text-zinc-200"
              >
                Reset
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Right panel: results */}
      <div className="flex-1 min-w-0">
        <HighlightList highlights={highlights} threshold={threshold} status={status} />
      </div>
    </div>
  );
}
