'use client';

import { createLogger } from '@/lib/logger';
import { useHistoryStore, calcDuration } from '@/store/historyStore';
import { useAppStore } from '@/store/appStore';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

const log = createLogger('RunsTable');

const STATUS_VARIANT: Record<string, 'success' | 'destructive' | 'info' | 'secondary'> = {
  done: 'success',
  error: 'destructive',
  running: 'info',
  pending: 'secondary',
};

function truncateUrl(url: string): string {
  try {
    const u = new URL(url);
    const v = u.searchParams.get('v');
    if (v) return `youtube.com/watch?v=${v}`;
    return u.hostname + u.pathname.slice(0, 30);
  } catch {
    return url.slice(0, 50);
  }
}

function formatDate(ms: number): string {
  try {
    return new Date(ms).toLocaleDateString('en-US', {
      day: '2-digit',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return String(ms);
  }
}

export function RunsTable() {
  const goUrl = useAppStore((s) => s.goUrl);
  const runs = useHistoryStore((s) => s.runs);
  const loading = useHistoryStore((s) => s.loading);
  const page = useHistoryStore((s) => s.page);
  const totalPages = useHistoryStore((s) => s.totalPages);
  const nextPage = useHistoryStore((s) => s.nextPage);
  const prevPage = useHistoryStore((s) => s.prevPage);
  const selectRun = useHistoryStore((s) => s.selectRun);
  const deleteRun = useHistoryStore((s) => s.deleteRun);
  const fetchRuns = useHistoryStore((s) => s.fetchRuns);
  const exportCSV = useHistoryStore((s) => s.exportCSV);

  const handleView = (id: number) => {
    log.info('viewing run detail', { id });
    void selectRun(goUrl, id);
  };

  const handleDelete = (id: number) => {
    const confirmed = window.confirm(`Delete run #${id}? This cannot be undone.`);
    if (!confirmed) return;
    log.info('deleting run', { id });
    void deleteRun(goUrl, id);
  };

  const handlePrev = () => {
    prevPage();
    void fetchRuns(goUrl);
  };

  const handleNext = () => {
    nextPage();
    void fetchRuns(goUrl);
  };

  return (
    <div className="flex flex-col gap-3">
      {/* Export CSV */}
      <div className="flex justify-end">
        <Button
          variant="outline"
          size="sm"
          onClick={exportCSV}
          disabled={runs.length === 0}
          className="text-subtle hover:text-heading"
        >
          Export CSV
        </Button>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-xl border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-background/60">
              <th className="px-3 py-2 text-left font-mono text-xs text-subtle">ID</th>
              <th className="px-3 py-2 text-left text-xs text-subtle">URL</th>
              <th className="px-3 py-2 text-left text-xs text-subtle">Mode</th>
              <th className="px-3 py-2 text-left text-xs text-subtle">Status</th>
              <th className="px-3 py-2 text-left text-xs text-subtle">Started</th>
              <th className="px-3 py-2 text-left text-xs text-subtle">Duration</th>
              <th className="px-3 py-2 text-right text-xs text-subtle">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <>
                {[0, 1, 2].map((i) => (
                  <tr key={i} className="border-b border-border/50">
                    {[0, 1, 2, 3, 4, 5, 6].map((j) => (
                      <td key={j} className="px-3 py-3">
                        <div className="h-3 rounded bg-surface animate-pulse w-full max-w-[120px]" />
                      </td>
                    ))}
                  </tr>
                ))}
              </>
            ) : runs.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-3 py-8 text-center text-sm text-subtle">
                  No pipeline runs found.
                </td>
              </tr>
            ) : (
              runs.map((run) => (
                <tr
                  key={run.id}
                  className="border-b border-border/50 hover:bg-background/40 transition-colors"
                >
                  <td className="px-3 py-2.5 font-mono text-xs text-subtle">
                    #{run.id}
                  </td>
                  <td className="px-3 py-2.5 max-w-[200px]">
                    <span className="block truncate text-heading">
                      {truncateUrl(run.url)}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-xs text-subtle">{run.mode}</td>
                  <td className="px-3 py-2.5">
                    <Badge variant={STATUS_VARIANT[run.status] ?? STATUS_VARIANT.pending}>
                      {run.status}
                    </Badge>
                  </td>
                  <td className="px-3 py-2.5 text-xs text-subtle whitespace-nowrap">
                    {formatDate(run.started_at)}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-xs text-subtle">
                    {calcDuration(run.started_at, run.finished_at)}
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center justify-end gap-1.5">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => handleView(run.id)}
                        className="h-auto px-3 py-1 text-xs text-subtle hover:text-heading"
                      >
                        View
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => handleDelete(run.id)}
                        aria-label={`Delete run #${run.id}`}
                        className="h-auto px-2 py-1 text-xs text-caption hover:border-destructive/40 hover:text-destructive"
                      >
                        ✕
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-center gap-4">
        <Button
          variant="outline"
          size="sm"
          onClick={handlePrev}
          disabled={page <= 1 || loading}
          className="text-subtle"
        >
          Previous
        </Button>
        <span className="text-xs text-subtle">
          Page {page} of {totalPages}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={handleNext}
          disabled={page >= totalPages || loading}
          className="text-subtle"
        >
          Next
        </Button>
      </div>
    </div>
  );
}
