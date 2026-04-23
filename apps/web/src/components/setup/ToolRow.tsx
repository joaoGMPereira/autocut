'use client';

import {
  CheckCircle2,
  XCircle,
  AlertCircle,
  Loader2,
  Download,
  ExternalLink,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import {
  AUTO_INSTALL_TOOLS,
  MANUAL_INSTALL_URLS,
  type ToolStatus,
} from '@/types/setup';

interface ToolRowProps {
  tool: ToolStatus;
}

export function ToolRow({ tool }: ToolRowProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const installStates = useSetupStore((s) => s.installStates);
  const installLogs = useSetupStore((s) => s.installLogs);
  const startInstall = useSetupStore((s) => s.startInstall);

  const state = installStates[tool.name] ?? 'idle';
  const logs = installLogs[tool.name] ?? [];

  const StatusIcon = () => {
    if (state === 'installing') {
      return <Loader2 className="h-5 w-5 text-info animate-spin shrink-0" />;
    }
    if (tool.installed) {
      return <CheckCircle2 className="h-5 w-5 text-success shrink-0" />;
    }
    if (tool.required) {
      return <XCircle className="h-5 w-5 text-destructive shrink-0" />;
    }
    return <AlertCircle className="h-5 w-5 text-warning shrink-0" />;
  };

  const badgeLabel = () => {
    if (state === 'installing') return 'Installing...';
    if (state === 'done') return 'Installed';
    if (state === 'error') return 'Error';
    if (tool.installed) return 'Installed';
    if (tool.required) return 'Missing';
    return 'Optional';
  };

  const badgeVariant = (): 'info' | 'success' | 'destructive' | 'warning' => {
    if (state === 'installing') return 'info';
    if (state === 'done' || tool.installed) return 'success';
    if (state === 'error' || tool.required) return 'destructive';
    return 'warning';
  };

  // All tools support Install() — auto-download (yt-dlp, Twitch) or copy from system PATH (others)
  const isAutoDownload = (AUTO_INSTALL_TOOLS as readonly string[]).includes(tool.name);
  const canInstall = !tool.installed && state === 'idle';
  const manualUrl = MANUAL_INSTALL_URLS[tool.name];
  // Show help link for copy-from-system tools so user knows the prerequisite
  const showManualLink = canInstall && !isAutoDownload && manualUrl;

  return (
    <div className="flex flex-col" data-testid={`tool-row-${tool.name}`}>
      <div className="flex items-center gap-3.5 px-5 py-3.5">
        <StatusIcon />

        <div className="flex flex-col gap-0.5 flex-1 min-w-0">
          <span className="text-sm font-semibold text-foreground">
            {tool.name}
          </span>
          {tool.installed && tool.path ? (
            <span className="text-[11px] font-mono text-subtle truncate">
              {tool.path}
              {tool.version ? ` \u00B7 ${tool.version}` : ''}
            </span>
          ) : (
            <span className="text-[11px] text-subtle">
              {state === 'installing'
                ? 'Installing...'
                : state === 'error'
                  ? 'Installation failed'
                  : tool.required
                    ? 'Not installed'
                    : 'Not installed (optional)'}
            </span>
          )}
        </div>

        {tool.updateAvailable && (
          <Badge variant="warning">↑ Update available</Badge>
        )}

        <Badge
          data-testid={`tool-status-${tool.name}`}
          variant={badgeVariant()}
        >
          {badgeLabel()}
        </Badge>

        {/* Check button removed — updates are checked automatically on gate mount */}

        {tool.updateAvailable && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 text-xs"
            onClick={() => startInstall(goUrl, tool.name)}
          >
            Update
          </Button>
        )}

        {canInstall && (
          <Button
            size="sm"
            data-testid={`tool-install-btn-${tool.name}`}
            className="h-7 gap-1.5 text-xs"
            onClick={() => startInstall(goUrl, tool.name)}
          >
            <Download className="h-3.5 w-3.5" />
            Install
          </Button>
        )}

        {showManualLink && (
          <a
            href={manualUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 text-xs font-medium text-subtle hover:text-foreground transition-colors shrink-0"
          >
            <ExternalLink className="h-3 w-3" />
            How to install
          </a>
        )}

        {state === 'error' && (
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

      {state === 'installing' && logs.length > 0 && (
        <div className="px-5 pb-3.5">
          <ScrollArea className="max-h-24 rounded-md bg-muted/50 p-2">
            <div className="flex flex-col gap-0.5">
              {logs.map((line, i) => (
                <span
                  key={i}
                  className="font-mono text-[11px] text-subtle"
                >
                  {line}
                </span>
              ))}
            </div>
          </ScrollArea>
        </div>
      )}
    </div>
  );
}
