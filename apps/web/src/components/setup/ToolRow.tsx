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
import { ScrollArea } from '@/components/ui/scroll-area';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import {
  AUTO_INSTALL_TOOLS,
  MANUAL_INSTALL_URLS,
  type ToolStatus,
} from '@/types/setup';
import { cn } from '@/lib/utils';

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
      return <Loader2 className="h-5 w-5 text-blue-400 animate-spin shrink-0" />;
    }
    if (tool.installed) {
      return <CheckCircle2 className="h-5 w-5 text-emerald-400 shrink-0" />;
    }
    if (tool.required) {
      return <XCircle className="h-5 w-5 text-red-400 shrink-0" />;
    }
    return <AlertCircle className="h-5 w-5 text-amber-400 shrink-0" />;
  };

  const badgeLabel = () => {
    if (state === 'installing') return 'Installing...';
    if (state === 'done') return 'Installed';
    if (state === 'error') return 'Error';
    if (tool.installed) return 'Installed';
    if (tool.required) return 'Missing';
    return 'Optional';
  };

  const badgeClass = () => {
    if (state === 'installing') return 'bg-blue-400/20 text-blue-400';
    if (state === 'done' || tool.installed)
      return 'bg-emerald-400/20 text-emerald-400';
    if (state === 'error') return 'bg-red-400/20 text-red-400';
    if (tool.required) return 'bg-red-400/20 text-red-400';
    return 'bg-amber-400/20 text-amber-400';
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
            <span className="text-[11px] font-mono text-muted-foreground truncate">
              {tool.path}
              {tool.version ? ` \u00B7 ${tool.version}` : ''}
            </span>
          ) : (
            <span className="text-[11px] text-muted-foreground">
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
          <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-400">
            ↑ Update available
          </span>
        )}

        <span
          data-testid={`tool-status-${tool.name}`}
          className={cn(
            'inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-semibold shrink-0',
            badgeClass(),
          )}
        >
          {badgeLabel()}
        </span>

        {/* Check button removed — updates are checked automatically on gate mount */}

        {tool.updateAvailable && (
          <Button
            size="sm"
            className="h-7 text-xs bg-[#00D4FF]/10 border border-[#00D4FF]/25 text-[#00D4FF] hover:bg-[#00D4FF]/20"
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
            className="inline-flex items-center gap-1.5 text-xs font-medium text-[#5c5c80] hover:text-muted-foreground transition-colors shrink-0"
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
  );
}
