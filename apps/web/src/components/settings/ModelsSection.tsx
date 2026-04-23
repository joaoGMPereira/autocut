'use client';

import { useEffect, useState } from 'react';
import { Download, Loader2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import { createLogger } from '@/lib/logger';

const log = createLogger('[ModelsSection]');

// ---------------------------------------------------------------------------
// Ollama types
// ---------------------------------------------------------------------------

interface OllamaModel {
  name: string;
  active: boolean;
}

interface OllamaStatus {
  available: boolean;
  models: OllamaModel[];
}

// ---------------------------------------------------------------------------
// ModelsSection
// ---------------------------------------------------------------------------

export function ModelsSection() {
  const goUrl = useAppStore((s) => s.goUrl);

  // Whisper state from setupStore
  const whisperModels = useSetupStore((s) => s.whisperModels);
  const fetchWhisperModels = useSetupStore((s) => s.fetchWhisperModels);
  const downloadWhisperModel = useSetupStore((s) => s.downloadWhisperModel);
  const whisperDownloadStates = useSetupStore((s) => s.whisperDownloadStates);
  const whisperDownloadLogs = useSetupStore((s) => s.whisperDownloadLogs);

  // Ollama state (fetched locally)
  const [ollamaStatus, setOllamaStatus] = useState<OllamaStatus | null>(null);
  const [ollamaLoading, setOllamaLoading] = useState(false);

  useEffect(() => {
    if (whisperModels.length === 0) {
      log.info('fetching whisper models on mount');
      void fetchWhisperModels(goUrl);
    }
  }, [whisperModels.length, goUrl, fetchWhisperModels]);

  useEffect(() => {
    const fetchOllama = async () => {
      setOllamaLoading(true);
      try {
        const res = await fetch(`${goUrl}/api/ollama/models`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = (await res.json()) as OllamaStatus;
        log.info('ollama status fetched', { available: data.available, count: data.models.length });
        setOllamaStatus(data);
      } catch (err) {
        log.error('failed to fetch ollama models', { err });
        setOllamaStatus({ available: false, models: [] });
      } finally {
        setOllamaLoading(false);
      }
    };
    void fetchOllama();
  }, [goUrl]);

  const activeDownload = Object.entries(whisperDownloadStates).find(
    ([, state]) => state === 'installing',
  )?.[0];

  return (
    <section className="space-y-6">
      {/* Whisper Models */}
      <div className="space-y-3">
        <h2 className="text-lg font-semibold text-foreground">Whisper Models</h2>
        <div className="rounded-xl border border-border bg-card">
          {whisperModels.length === 0 ? (
            <div className="px-5 py-8 text-center text-sm text-subtle">
              Loading models…
            </div>
          ) : (
            whisperModels.map((model, i) => {
              const state = whisperDownloadStates[model.name];
              const isDownloading = state === 'installing';

              return (
                <div key={model.name}>
                  {i > 0 && <Separator />}
                  <div className="flex items-center justify-between px-5 py-3">
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-medium text-foreground capitalize">
                        {model.name}
                      </span>
                      <span className="text-xs text-subtle">{model.size_mb} MB</span>
                      {model.active && (
                        <Badge variant="secondary" className="text-[10px] h-4 px-1.5">
                          Active
                        </Badge>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      {model.downloaded ? (
                        <Badge variant="success" className="text-[10px] h-5 px-2">
                          Downloaded
                        </Badge>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-7 gap-1.5 text-xs"
                          disabled={isDownloading || !!activeDownload}
                          onClick={() => {
                            log.info('downloading whisper model', { model: model.name });
                            downloadWhisperModel(goUrl, model.name);
                          }}
                        >
                          {isDownloading ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <Download className="h-3 w-3" />
                          )}
                          {isDownloading ? 'Downloading…' : 'Download'}
                        </Button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Download log */}
        {activeDownload && (whisperDownloadLogs[activeDownload] ?? []).length > 0 && (
          <div className="rounded-xl border border-border bg-card px-4 py-3 space-y-2">
            <p className="text-xs font-semibold text-subtle uppercase tracking-wide">
              Downloading {activeDownload}…
            </p>
            <div className="max-h-32 overflow-y-auto flex flex-col gap-0.5">
              {(whisperDownloadLogs[activeDownload] ?? []).map((line, i) => (
                <span key={i} className="font-mono text-[11px] text-subtle">
                  {line}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Ollama Models */}
      <div className="space-y-3">
        <h2 className="text-lg font-semibold text-foreground">Ollama Models</h2>
        <div className="rounded-xl border border-border bg-card">
          {ollamaLoading ? (
            <div className="flex items-center gap-2 px-5 py-8 text-sm text-subtle">
              <Loader2 className="h-4 w-4 animate-spin" />
              Connecting to Ollama…
            </div>
          ) : !ollamaStatus?.available ? (
            <div className="px-5 py-8 text-center space-y-1">
              <p className="text-sm font-medium text-foreground">Ollama not available</p>
              <p className="text-xs text-subtle">
                Start Ollama or install it from{' '}
                <span className="font-mono text-xs">ollama.ai</span>
              </p>
            </div>
          ) : ollamaStatus.models.length === 0 ? (
            <div className="px-5 py-8 text-center text-sm text-subtle">
              No models installed. Run{' '}
              <span className="font-mono text-xs">ollama pull &lt;model&gt;</span> to add one.
            </div>
          ) : (
            ollamaStatus.models.map((model, i) => (
              <div key={model.name}>
                {i > 0 && <Separator />}
                <div className="flex items-center justify-between px-5 py-3">
                  <span className="text-sm font-medium text-foreground font-mono">
                    {model.name}
                  </span>
                  {model.active && (
                    <Badge variant="secondary" className="text-[10px] h-4 px-1.5">
                      Active
                    </Badge>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </section>
  );
}
