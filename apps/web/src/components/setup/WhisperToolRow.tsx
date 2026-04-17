'use client';

import { CheckCircle2, XCircle, AlertCircle, Loader2, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import type { ToolStatus } from '@/types/setup';
import { cn } from '@/lib/utils';

interface WhisperToolRowProps {
  tool: ToolStatus;
}

export function WhisperToolRow({ tool }: WhisperToolRowProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const installStates = useSetupStore((s) => s.installStates);
  const installLogs = useSetupStore((s) => s.installLogs);
  const whisperModels = useSetupStore((s) => s.whisperModels);
  const whisperDownloadStates = useSetupStore((s) => s.whisperDownloadStates);
  const whisperDownloadLogs = useSetupStore((s) => s.whisperDownloadLogs);
  const downloadWhisperModel = useSetupStore((s) => s.downloadWhisperModel);

  const installState = installStates[tool.name] ?? 'idle';
  const installLog = installLogs[tool.name] ?? [];

  // Binary detection state
  const binaryInstalling = installState === 'installing';
  // "no_model" means binary found, model missing. Anything else with installed=false = binary missing.
  const binaryPresent = tool.installed || tool.source === 'no_model';
  const hasModel = tool.installed; // only true when binary + model both present
  const binaryOnly = binaryPresent && !hasModel;

  const activeDownload = Object.entries(whisperDownloadStates).find(
    ([, s]) => s === 'installing',
  )?.[0];

  const downloadedModels = whisperModels.filter((m) => m.downloaded);

  return (
    <div className="flex flex-col" data-testid="tool-row-whisper-cli">
      {/* ── Binary row ── */}
      <div className="flex items-center gap-3.5 px-5 py-3.5">
        {binaryInstalling ? (
          <Loader2 className="h-5 w-5 text-blue-400 animate-spin shrink-0" />
        ) : hasModel ? (
          <CheckCircle2 className="h-5 w-5 text-emerald-400 shrink-0" />
        ) : binaryOnly ? (
          <AlertCircle className="h-5 w-5 text-amber-400 shrink-0" />
        ) : (
          <XCircle className="h-5 w-5 text-red-400 shrink-0" />
        )}

        <div className="flex flex-col gap-0.5 flex-1 min-w-0">
          <span className="text-sm font-semibold text-foreground">whisper-cli</span>
          {binaryPresent && tool.path ? (
            <span className="text-[11px] font-mono text-muted-foreground truncate">
              {tool.path}
              {tool.version ? ` · ${tool.version}` : ''}
            </span>
          ) : (
            <span className="text-[11px] text-muted-foreground">
              {binaryInstalling ? 'Installing…' : 'Not installed'}
            </span>
          )}
        </div>

        <span
          data-testid="tool-status-whisper-cli"
          className={cn(
            'inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold shrink-0',
            binaryInstalling
              ? 'bg-blue-400/20 text-blue-400'
              : hasModel
                ? 'bg-emerald-400/20 text-emerald-400'
                : binaryOnly
                  ? 'bg-amber-400/20 text-amber-400'
                  : 'bg-red-400/20 text-red-400',
          )}
        >
          {binaryInstalling
            ? 'Installing…'
            : hasModel
              ? 'Installed'
              : binaryOnly
                ? 'Missing model'
                : 'Missing'}
        </span>
      </div>

      {/* install log for binary */}
      {binaryInstalling && installLog.length > 0 && (
        <div className="px-5 pb-3.5">
          <ScrollArea className="max-h-24 rounded-md bg-muted/50 p-2">
            <div className="flex flex-col gap-0.5">
              {installLog.map((line, i) => (
                <span key={i} className="font-mono text-[11px] text-muted-foreground">
                  {line}
                </span>
              ))}
            </div>
          </ScrollArea>
        </div>
      )}

      {/* ── Model sub-section (only shown when binary is present) ── */}
      {binaryPresent && (
        <>
          <Separator />
          <div className="px-5 py-3 space-y-2.5">
            {/* Sub-section header */}
            <div className="flex items-center gap-2">
              {hasModel ? (
                <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0" />
              ) : (
                <AlertCircle className="h-4 w-4 text-red-400 shrink-0" />
              )}
              <span className="text-xs font-semibold text-foreground">
                Model
              </span>
              <span
                className={cn(
                  'inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-semibold',
                  hasModel
                    ? 'bg-emerald-400/20 text-emerald-400'
                    : 'bg-red-400/20 text-red-400',
                )}
              >
                {hasModel
                  ? `${downloadedModels.length} downloaded`
                  : 'Required — none downloaded'}
              </span>
            </div>

            {/* Model list */}
            {whisperModels.length === 0 ? (
              <div className="text-[11px] text-muted-foreground pl-6">
                Loading models…
              </div>
            ) : (
              <div className="pl-6 space-y-1.5">
                {whisperModels.map((model) => {
                  const dlState = whisperDownloadStates[model.name];
                  const isDownloading = dlState === 'installing';
                  const dlLogs = whisperDownloadLogs[model.name] ?? [];

                  return (
                    <div key={model.name} className="flex flex-col gap-1">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="text-[12px] font-medium text-foreground capitalize">
                            {model.name}
                          </span>
                          <span className="text-[11px] text-muted-foreground">
                            {model.size_mb} MB
                          </span>
                          {model.active && (
                            <span className="inline-flex items-center rounded-full bg-blue-400/20 px-1.5 py-0.5 text-[10px] font-semibold text-blue-400">
                              Active
                            </span>
                          )}
                        </div>

                        {model.downloaded ? (
                          <span className="inline-flex items-center rounded-full border border-emerald-500/40 px-2 py-0.5 text-[10px] font-medium text-emerald-400">
                            Downloaded
                          </span>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-6 gap-1 text-[11px] px-2"
                            disabled={isDownloading || !!activeDownload}
                            onClick={() => downloadWhisperModel(goUrl, model.name)}
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

                      {isDownloading && dlLogs.length > 0 && (
                        <ScrollArea className="max-h-16 rounded bg-muted/50 p-1.5">
                          <div className="flex flex-col gap-0.5">
                            {dlLogs.map((line, i) => (
                              <span key={i} className="font-mono text-[10px] text-muted-foreground">
                                {line}
                              </span>
                            ))}
                          </div>
                        </ScrollArea>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
