'use client';

import { useEffect, useState } from 'react';
import { RefreshCw, Folder } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useAppStore } from '@/store/appStore';
import { useSetupStore } from '@/store/setupStore';
import { ToolRow } from '@/components/setup/ToolRow';
import { createLogger } from '@/lib/logger';

const log = createLogger('SettingsPage');

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
  const [dirInfo, setDirInfo] = useState<DirInfo | null>(null);

  useEffect(() => {
    if (tools.length === 0) {
      fetchStatus(goUrl);
    }
  }, [tools.length, goUrl, fetchStatus]);

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
