'use client';

import { useEffect } from 'react';
import Link from 'next/link';
import { Download } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useStatsStore } from '@/store/statsStore';
import { createLogger } from '@/lib/logger';
import type { PipelineRunSummary } from '@/store/statsStore';

const log = createLogger('DashboardClient');

const quickActions = [
  { label: 'New Download', href: '/pipeline' },
] as const;

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'done':
      return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
    case 'error':
      return 'bg-red-500/10 text-red-400 border-red-500/20';
    case 'running':
      return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
    default:
      return 'bg-zinc-500/10 text-zinc-400 border-zinc-500/20';
  }
}

function formatTimestamp(ms: number): string {
  return new Date(ms).toLocaleString();
}

function formatDuration(run: PipelineRunSummary): string {
  if (run.finished_at === null) return '—';
  const secs = ((run.finished_at - run.started_at) / 1000).toFixed(0);
  return `${secs}s`;
}

export default function DashboardClient() {
  const { stats, loading, fetchStats } = useStatsStore();

  useEffect(() => {
    log.info('mounting — initial fetch');
    fetchStats();
    const interval = setInterval(() => {
      log.debug('auto-refresh tick');
      fetchStats();
    }, 30_000);
    return () => {
      log.debug('unmounting — clearing interval');
      clearInterval(interval);
    };
  }, [fetchStats]);

  const activeDownloads = stats?.active_downloads ?? 0;
  const recentRuns = stats?.recent_pipeline_runs ?? [];

  return (
    <div className="flex flex-col gap-6">
      {/* Stats — only validated features */}
      <div className="grid grid-cols-3 gap-4">
        <Card className="border-border bg-card">
          <CardContent className="flex items-center gap-4 p-5">
            <Download className="h-5 w-5 text-brand shrink-0" />
            <div className="min-w-0">
              <p className="text-xs text-muted-foreground leading-none mb-1">Active Downloads</p>
              <p className="font-mono text-2xl font-semibold tabular-nums leading-none">
                {loading && stats === null ? '…' : activeDownloads}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Pipeline Runs */}
      <div className="flex flex-col gap-3">
        <h2 className="text-[16px] font-semibold">Recent Runs</h2>
        <div className="rounded-xl border border-border overflow-hidden">
          {recentRuns.length === 0 ? (
            <p className="text-sm text-muted-foreground px-5 py-4">
              {loading && stats === null ? 'Loading…' : 'No pipeline runs yet.'}
            </p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-card/50">
                  <th className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">ID</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">Mode</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">Status</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">Started</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">Duration</th>
                </tr>
              </thead>
              <tbody>
                {recentRuns.map((run) => (
                  <tr key={run.id} className="border-b border-border last:border-0 hover:bg-card/30 transition-colors">
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">#{run.id}</td>
                    <td className="px-4 py-3 text-xs capitalize">{run.mode}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs border ${statusBadgeClass(run.status)}`}>
                        {run.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">{formatTimestamp(run.started_at)}</td>
                    <td className="px-4 py-3 font-mono text-xs">{formatDuration(run)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Quick Actions */}
      <div className="flex flex-col gap-3">
        <h2 className="text-[16px] font-semibold">Quick Actions</h2>
        <div className="grid grid-cols-3 gap-3">
          {quickActions.map(({ label, href }) => (
            <Button key={href} variant="outline" asChild className="h-11">
              <Link href={href}>{label}</Link>
            </Button>
          ))}
        </div>
      </div>
    </div>
  );
}
