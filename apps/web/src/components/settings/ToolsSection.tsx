'use client';

import { useEffect, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import { ToolRow } from '@/components/setup/ToolRow';
import { WhisperToolRow } from '@/components/setup/WhisperToolRow';
import { CustomPathDialog } from '@/components/setup/CustomPathDialog';
import { AUTO_INSTALL_TOOLS } from '@/types/setup';
import { createLogger } from '@/lib/logger';

const log = createLogger('ToolsSection');

export function ToolsSection() {
  const goUrl = useAppStore((s) => s.goUrl);
  const tools = useSetupStore((s) => s.tools);
  const [dialogTool, setDialogTool] = useState<string | null>(null);
  const loading = useSetupStore((s) => s.loading);
  const fetchStatus = useSetupStore((s) => s.fetchStatus);
  const checkUpdate = useSetupStore((s) => s.checkUpdate);
  const fetchWhisperModels = useSetupStore((s) => s.fetchWhisperModels);

  useEffect(() => {
    if (tools.length === 0) {
      log.info('fetching tools status on mount');
      void fetchStatus(goUrl);
    }
    void fetchWhisperModels(goUrl);
    const installedAutoTools = tools.filter(
      (t) => t.installed && (AUTO_INSTALL_TOOLS as readonly string[]).includes(t.name),
    );
    for (const t of installedAutoTools) {
      void checkUpdate(goUrl, t.name);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">Tools</h2>
        <Button
          variant="outline"
          size="sm"
          className="h-7 gap-1.5 text-xs"
          disabled={loading}
          onClick={() => {
            log.info('re-checking tool status');
            void fetchStatus(goUrl);
          }}
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
          Re-check
        </Button>
      </div>

      <div className="rounded-xl border border-border bg-card">
        {tools.map((tool, i) => (
          <div key={tool.name}>
            {i > 0 && <Separator />}
            {tool.name === 'whisper-cli' ? (
              <WhisperToolRow tool={tool} />
            ) : (
              <ToolRow tool={tool} />
            )}
            <div className="px-5 pb-2">
              <button
                className="text-xs text-subtle hover:text-foreground underline-offset-2 hover:underline"
                onClick={() => setDialogTool(tool.name)}
              >
                {tool.source === 'custom' ? 'Edit custom path' : 'Set custom path'}
              </button>
            </div>
          </div>
        ))}
        {tools.length === 0 && !loading && (
          <div className="px-5 py-8 text-center text-sm text-subtle">
            Could not fetch tool status. Is the backend running?
          </div>
        )}
      </div>

      {dialogTool && (
        <CustomPathDialog
          toolName={dialogTool}
          currentCustomPath={tools.find(t => t.name === dialogTool)?.source === 'custom'
            ? tools.find(t => t.name === dialogTool)?.path
            : undefined}
          open={dialogTool !== null}
          onOpenChange={(open) => { if (!open) setDialogTool(null); }}
        />
      )}
    </section>
  );
}
