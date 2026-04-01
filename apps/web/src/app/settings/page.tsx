'use client';

import { useEffect, useState } from 'react';
import { RefreshCw, Folder, Download, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import { ToolRow } from '@/components/setup/ToolRow';
import { createLogger } from '@/lib/logger';
import type { WhisperModelInfo } from '@/types/setup';

const log = createLogger('SettingsPage');

const WHISPER_MODELS_FALLBACK: Omit<WhisperModelInfo, 'downloaded' | 'path'>[] = [
  { name: 'tiny', size_mb: 75 },
  { name: 'base', size_mb: 142 },
  { name: 'small', size_mb: 466 },
  { name: 'medium', size_mb: 1500 },
  { name: 'large', size_mb: 2900 },
];

interface DirInfo {
  root: string;
  bin_dir: string;
  models_dir: string;
  tokens_dir: string;
  cache_dir: string;
  downloads_dir: string;
  thumbnails_dir: string;
}

const DIR_LABELS: { key: keyof Omit<DirInfo, 'root'>; label: string }[] = [
  { key: 'bin_dir', label: 'bin/' },
  { key: 'downloads_dir', label: 'downloads/' },
  { key: 'models_dir', label: 'models/' },
  { key: 'thumbnails_dir', label: 'thumbnails/' },
  { key: 'cache_dir', label: 'cache/' },
  { key: 'tokens_dir', label: 'tokens/' },
];

export default function SettingsPage() {
  const goUrl = useAppStore((s) => s.goUrl);
  const tools = useSetupStore((s) => s.tools);
  const loading = useSetupStore((s) => s.loading);
  const fetchStatus = useSetupStore((s) => s.fetchStatus);
  const whisperModels = useSetupStore((s) => s.whisperModels);
  const whisperDownloadStates = useSetupStore((s) => s.whisperDownloadStates);
  const whisperDownloadLogs = useSetupStore((s) => s.whisperDownloadLogs);
  const fetchWhisperModels = useSetupStore((s) => s.fetchWhisperModels);
  const downloadWhisperModel = useSetupStore((s) => s.downloadWhisperModel);
  const whisperInstalled = tools.some((t) => t.name === 'whisper' && t.installed);
  const [dirInfo, setDirInfo] = useState<DirInfo | null>(null);

  useEffect(() => {
    if (tools.length === 0) {
      fetchStatus(goUrl);
    }
  }, [tools.length, goUrl, fetchStatus]);

  useEffect(() => {
    fetchWhisperModels(goUrl);
  }, [goUrl, fetchWhisperModels]);

  useEffect(() => {
    fetch(`${goUrl}/api/setup/dir`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data) setDirInfo(data);
      })
      .catch((err) => {
        log.error('failed to fetch dir info', { err });
      });
  }, [goUrl]);

  return (
    <div className="p-6 max-w-3xl space-y-8">
      <h1 className="text-2xl font-semibold text-foreground">Settings</h1>

      {/* Tools Section */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-foreground">Tools</h2>
          <Button
            variant="outline"
            size="sm"
            className="h-7 gap-1.5 text-xs"
            disabled={loading}
            onClick={() => fetchStatus(goUrl)}
          >
            <RefreshCw
              className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`}
            />
            Re-check
          </Button>
        </div>

        <div className="rounded-xl border border-border bg-card">
          {tools.length > 0 ? (
            tools.map((tool, i) => (
              <div key={tool.name}>
                {i > 0 && <Separator />}
                <ToolRow tool={tool} />
              </div>
            ))
          ) : (
            <div className="px-5 py-8 text-center text-sm text-muted-foreground">
              {loading
                ? 'Checking tool status...'
                : 'Could not fetch tool status. Is the backend running?'}
            </div>
          )}
        </div>
      </section>

      {/* Whisper Models Section */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-foreground">
            Whisper Models
          </h2>
          <Button
            variant="outline"
            size="sm"
            className="h-7 gap-1.5 text-xs"
            onClick={() => fetchWhisperModels(goUrl)}
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Refresh
          </Button>
        </div>

        <div className="rounded-xl border border-border bg-card">
          {(whisperModels.length > 0
            ? whisperModels
            : WHISPER_MODELS_FALLBACK.map((m) => ({
                ...m,
                downloaded: false,
                path: '',
              }))
          ).map((model, i, arr) => {
            const dlState = whisperDownloadStates[model.name] ?? 'idle';
            const dlLogs = whisperDownloadLogs[model.name] ?? [];
            return (
              <div key={model.name}>
                {i > 0 && <Separator />}
                <div className="flex flex-col">
                  <div className="flex items-center gap-3.5 px-5 py-3.5">
                    <div className="flex flex-col gap-0.5 flex-1 min-w-0">
                      <span className="text-[13px] font-semibold text-foreground">
                        {model.name}
                      </span>
                      <span className="text-[11px] font-mono text-muted-foreground truncate">
                        {model.downloaded
                          ? model.path
                          : `${model.size_mb} MB`}
                      </span>
                    </div>

                    {dlState === 'installing' ? (
                      <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-blue-400/20 text-blue-400">
                        <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                        Downloading...
                      </span>
                    ) : model.downloaded ? (
                      <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-emerald-400/20 text-emerald-400">
                        Downloaded
                      </span>
                    ) : (
                      <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold bg-muted text-muted-foreground">
                        Available
                      </span>
                    )}

                    {!model.downloaded && dlState !== 'installing' && (
                      <Button
                        size="sm"
                        className="h-7 gap-1.5 text-xs bg-[#1e1e2e] border border-[#2a2a3a] text-[#8888a8] hover:bg-[#252535]"
                        onClick={() => downloadWhisperModel(goUrl, model.name)}
                      >
                        <Download className="h-3.5 w-3.5" />
                        Download
                      </Button>
                    )}
                  </div>

                  {dlState === 'installing' && dlLogs.length > 0 && (
                    <div className="px-5 pb-3.5">
                      <ScrollArea className="h-16 rounded-md bg-muted/50 p-2">
                        <div className="flex flex-col gap-0.5">
                          {dlLogs.map((line, j) => (
                            <span
                              key={j}
                              className="font-mono text-[11px] text-muted-foreground"
                            >
                              {line}
                            </span>
                          ))}
                        </div>
                      </ScrollArea>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>

        {!whisperInstalled && (
          <p className="text-xs text-muted-foreground">
            whisper must be installed to use models
          </p>
        )}
      </section>

      <Separator />

      {/* Data Directory Section */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold text-foreground">
          Data Directory
        </h2>

        <div className="rounded-xl border border-border bg-card p-5 space-y-4">
          {dirInfo ? (
            <>
              <div className="flex items-center gap-2">
                <Folder className="h-4 w-4 text-blue-400" />
                <span className="font-mono text-sm font-medium text-foreground">
                  {dirInfo.root}
                </span>
              </div>
              <div className="grid grid-cols-2 gap-2">
                {DIR_LABELS.map(({ key, label }) => (
                  <div key={key} className="flex items-center gap-2">
                    <Folder className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="font-mono text-xs text-muted-foreground">
                      {label}
                    </span>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="text-sm text-muted-foreground">
              Loading directory info...
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
