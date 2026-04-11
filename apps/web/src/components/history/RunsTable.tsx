'use client';

import { createLogger } from '@/lib/logger';
import { useHistoryStore, calcDuration } from '@/store/historyStore';
import { useAppStore } from '@/store/appStore';
import { Button } from '@/components/ui/button';

const log = createLogger('RunsTable');

const STATUS_CLASSES: Record<string, string> = {
  done: 'text-emerald-400 border-emerald-500/30 bg-emerald-500/5',
  error: 'text-red-400 border-red-400/30 bg-red-400/5',
  running: 'text-blue-400 border-blue-500/30 bg-blue-500/5',
  pending: 'text-zinc-400 border-zinc-600 bg-zinc-800/50',
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
          className="text-zinc-400 hover:text-[#F0F0F8]"
        >
          Export CSV
        </Button>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-xl border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-zinc-900/60">
              <th className="px-3 py-2 text-left font-mono text-xs text-zinc-500">ID</th>
              <th className="px-3 py-2 text-left text-xs text-zinc-500">URL</th>
              <th className="px-3 py-2 text-left text-xs text-zinc-500">Mode</th>
              <th className="px-3 py-2 text-left text-xs text-zinc-500">Status</th>
              <th className="px-3 py-2 text-left text-xs text-zinc-500">Started</th>
              <th className="px-3 py-2 text-left text-xs text-zinc-500">Duration</th>
              <th className="px-3 py-2 text-right text-xs text-zinc-500">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <>
                {[0, 1, 2].map((i) => (
                  <tr key={i} className="border-b border-border/50">
                    {[0, 1, 2, 3, 4, 5, 6].map((j) => (
                      <td key={j} className="px-3 py-3">
                        <div className="h-3 rounded bg-zinc-800 animate-pulse w-full max-w-[120px]" />
                      </td>
                    ))}
                  </tr>
                ))}
              </>
            ) : runs.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-3 py-8 text-center text-sm text-zinc-500">
                  No pipeline runs found.
                </td>
              </tr>
            ) : (
              runs.map((run) => {
                const statusClass = STATUS_CLASSES[run.status] ?? STATUS_CLASSES.pending;
                return (
                  <tr
                    key={run.id}
                    className="border-b border-border/50 hover:bg-zinc-900/40 transition-colors"
                  >
                    <td className="px-3 py-2.5 font-mono text-xs text-zinc-500">
                      #{run.id}
                    </td>
                    <td className="px-3 py-2.5 max-w-[200px]">
                      <span className="block truncate text-[#F0F0F8]">
                        {truncateUrl(run.url)}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-xs text-zinc-400">{run.mode}</td>
                    <td className="px-3 py-2.5">
                      <span
                        className={`inline-block rounded-md border px-2 py-0.5 text-xs ${statusClass}`}
                      >
                        {run.status}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-xs text-zinc-400 whitespace-nowrap">
                      {formatDate(run.started_at)}
                    </td>
                    <td className="px-3 py-2.5 font-mono text-xs text-zinc-500">
                      {calcDuration(run.started_at, run.finished_at)}
                    </td>
                    <td className="px-3 py-2.5">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          type="button"
                          onClick={() => handleView(run.id)}
                          className="rounded-md border border-border bg-zinc-800 px-3 py-1 text-xs text-zinc-400 hover:border-zinc-500 hover:text-[#F0F0F8] transition-colors"
                        >
                          View
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(run.id)}
                          aria-label={`Delete run #${run.id}`}
                          className="rounded-md border border-border bg-zinc-800 px-2 py-1 text-xs text-zinc-600 hover:border-red-400/40 hover:text-red-400 transition-colors"
                        >
                          ✕
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })
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
          className="text-zinc-400"
        >
          Previous
        </Button>
        <span className="text-xs text-zinc-500">
          Page {page} of {totalPages}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={handleNext}
          disabled={page >= totalPages || loading}
          className="text-zinc-400"
        >
          Next
        </Button>
      </div>
    </div>
  );
}
