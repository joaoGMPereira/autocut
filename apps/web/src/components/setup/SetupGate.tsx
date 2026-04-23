'use client';

import { useEffect } from 'react';
import { RefreshCw, Scissors, FolderSync } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import { ToolRow } from './ToolRow';
import { WhisperToolRow } from './WhisperToolRow';
import { AUTO_INSTALL_TOOLS, type ToolStatus } from '@/types/setup';

interface SetupGateProps {
  tools: ToolStatus[];
  onComplete: () => void;
}

export function SetupGate({ tools, onComplete }: SetupGateProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const allRequiredInstalled = useSetupStore((s) => s.allRequiredInstalled);
  const loading = useSetupStore((s) => s.loading);
  const fetchStatus = useSetupStore((s) => s.fetchStatus);
  const syncTools = useSetupStore((s) => s.syncTools);
  const checkUpdate = useSetupStore((s) => s.checkUpdate);
  const fetchWhisperModels = useSetupStore((s) => s.fetchWhisperModels);
  const isReady = allRequiredInstalled();

  useEffect(() => {
    // Fetch whisper models so the sub-section can show them
    void fetchWhisperModels(goUrl);

    // Auto-check updates for installed auto-downloadable tools
    const installedAutoTools = tools.filter(
      (t) => t.installed && (AUTO_INSTALL_TOOLS as readonly string[]).includes(t.name),
    );
    for (const t of installedAutoTools) {
      void checkUpdate(goUrl, t.name);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="flex h-full items-center justify-center" data-testid="setup-gate">
      <div className="flex flex-col items-center gap-8 w-full max-w-[600px] px-4">
        {/* Header */}
        <div className="flex flex-col items-center gap-4">
          <Scissors className="h-14 w-14 text-brand" />
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            AutoCut Setup
          </h1>
          <p className="text-sm text-subtle text-center max-w-[480px]">
            Some required tools need to be installed before you can use AutoCut.
          </p>
        </div>

        {/* Tool list */}
        <div className="w-full rounded-xl border border-border bg-card">
          {tools.map((tool, i) => (
            <div key={tool.name}>
              {i > 0 && <Separator />}
              {tool.name === 'whisper-cli' ? (
                <WhisperToolRow tool={tool} />
              ) : (
                <>
                  <ToolRow tool={tool} />
                  {tool.versionOk === false && (
                    <div className="px-5 pb-2 flex items-center gap-2">
                      <Badge variant="warning">Version too old</Badge>
                      <span className="text-xs text-subtle">
                        Update required for full compatibility
                      </span>
                    </div>
                  )}
                </>
              )}
            </div>
          ))}
        </div>

        {/* Footer */}
        <div className="flex flex-col items-center gap-3 w-full">
          <Button
            data-testid="setup-continue-btn"
            className="w-full h-11 text-sm font-semibold"
            disabled={!isReady}
            onClick={onComplete}
          >
            Continue
          </Button>
          <div className="flex gap-2">
            <Button
              data-testid="setup-sync-btn"
              variant="outline"
              size="sm"
              className="gap-1.5 text-xs"
              disabled={loading}
              onClick={() => void syncTools(goUrl)}
            >
              <FolderSync className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              Sync from computer
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="gap-1.5 text-xs"
              disabled={loading}
              onClick={() => void fetchStatus(goUrl)}
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              Re-check
            </Button>
          </div>
          {!isReady && (
            <p className="text-xs text-subtle">
              Install all required tools to continue
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
