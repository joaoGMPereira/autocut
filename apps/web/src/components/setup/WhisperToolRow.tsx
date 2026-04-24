'use client';

import { CheckCircle2, XCircle, AlertCircle, Loader2, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import type { ToolStatus } from '@/types/setup';

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

  const startInstall = useSetupStore((s) => s.startInstall);

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
          <Loader2 className="h-5 w-5 text-info animate-spin shrink-0" />
        ) : hasModel ? (
          <CheckCircle2 className="h-5 w-5 text-success shrink-0" />
        ) : binaryOnly ? (
          <AlertCircle className="h-5 w-5 text-warning shrink-0" />
        ) : (
          <XCircle className="h-5 w-5 text-destructive shrink-0" />
        )}

        <div className="flex flex-col gap-0.5 flex-1 min-w-0">
          <span className="text-sm font-semibold text-foreground">whisper-cli</span>
          {binaryPresent && tool.path ? (
            <span className="text-[11px] font-mono text-subtle truncate">
              {tool.path}
              {tool.version ? ` · ${tool.version}` : ''}
            </span>
          ) : (
            <span className="text-[11px] text-subtle">
              {binaryInstalling ? 'Installing…' : 'Not installed'}
            </span>
          )}
        </div>

        <Badge
          data-testid="tool-status-whisper-cli"
          variant={
            binaryInstalling
              ? 'info'
              : hasModel
                ? 'success'
                : binaryOnly
                  ? 'warning'
                  : installState === 'error'
                    ? 'destructive'
                    : 'destructive'
          }
        >
          {binaryInstalling
            ? 'Installing…'
            : hasModel
              ? 'Installed'
              : binaryOnly
                ? 'Missing model'
                : installState === 'error'
                  ? 'Error'
                  : 'Missing'}
        </Badge>

        {!binaryPresent && installState === 'idle' && (
          <Button
            size="sm"
            data-testid="tool-install-btn-whisper-cli"
            className="h-7 gap-1.5 text-xs"
            onClick={() => startInstall(goUrl, tool.name)}
          >
            <Download className="h-3.5 w-3.5" />
            Install
          </Button>
        )}

        {!binaryPresent && installState === 'error' && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 text-xs"
            onClick={() => startInstall(goUrl, tool.name)}
          >
            Retry
          </Button>
        )}
      </div>

      {/* install log for binary */}
      {(binaryInstalling || installState === 'error') && installLog.length > 0 && (
        <div className="px-5 pb-3.5">
          <div className="max-h-24 overflow-y-auto rounded-md bg-muted/50 p-2 flex flex-col gap-0.5">
            {installLog.map((line, i) => (
              <span key={i} className="font-mono text-[11px] text-subtle">
                {line}
              </span>
            ))}
          </div>
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
                <CheckCircle2 className="h-4 w-4 text-success shrink-0" />
              ) : (
                <AlertCircle className="h-4 w-4 text-destructive shrink-0" />
              )}
              <span className="text-xs font-semibold text-foreground">
                Model
              </span>
              <Badge variant={hasModel ? 'success' : 'destructive'}>
                {hasModel
                  ? `${downloadedModels.length} downloaded`
                  : 'Required — none downloaded'}
              </Badge>
            </div>

            {/* Model list */}
            {whisperModels.length === 0 ? (
              <div className="text-[11px] text-subtle pl-6">
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
                          <span className="text-[11px] text-subtle">
                            {model.size_mb} MB
                          </span>
                          {model.active && (
                            <Badge variant="info">Active</Badge>
                          )}
                        </div>

                        {model.downloaded ? (
                          <Badge variant="success">Downloaded</Badge>
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
                        <div className="max-h-16 overflow-y-auto rounded bg-muted/50 p-1.5 flex flex-col gap-0.5">
                          {dlLogs.map((line, i) => (
                            <span key={i} className="font-mono text-[10px] text-subtle">
                              {line}
                            </span>
                          ))}
                        </div>
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
