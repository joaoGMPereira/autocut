'use client';

import { useState } from 'react';
import { Separator } from '@/components/ui/separator';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
        <div className="rounded-xl border border-border bg-card p-5 flex flex-col gap-5">
          {/* Video path */}
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs font-medium text-subtle">Caminho do Vídeo</Label>
            <Input
              type="text"
              value={videoPath}
              onChange={(e) => setVideoPath(e.target.value)}
              placeholder="/path/to/video.mp4"
              disabled={isRunning}
            />
          </div>

          <Separator />

          {/* Strategy selector */}
          <HighlightStrategySelector
            selected={strategies}
            chatJsonPath={chatJsonPath}
            onToggle={toggleStrategy}
            onChatJsonPathChange={setChatJsonPath}
          />

          <Separator />

          {/* Threshold slider */}
          <ConfidenceThresholdSlider value={threshold} onChange={setThreshold} />

          {/* Error */}
          {error && (
            <p className="text-xs text-destructive">{error}</p>
          )}

          {/* Actions */}
          <div className="flex gap-2">
            <Button
              variant="brand"
              onClick={handleRun}
              disabled={isRunning || !videoPath.trim() || strategies.length === 0}
              className="flex-1 disabled:opacity-50"
            >
              {isRunning ? 'Detectando…' : 'Detect Highlights'}
            </Button>
            {(status === 'done' || status === 'error') && (
              <Button
                variant="outline"
                onClick={handleReset}
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
