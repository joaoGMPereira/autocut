'use client';

import Image from 'next/image';
import { cn } from '@/lib/utils';
import type { PipelineRun } from '@/store/pipelineStore';

interface RecentRunsSectionProps {
  runs: PipelineRun[];
  onResume: (runId: number) => void;
  onDelete: (runId: number) => void;
}

const STATUS_LABELS: Record<string, string> = {
  pending: 'Pendente',
  in_progress: 'Em andamento',
  completed: 'Concluído',
  failed: 'Falhou',
};

const STATUS_COLORS: Record<string, string> = {
  pending: 'text-[#A0A0B8] border-[#A0A0B8]/30 bg-[#A0A0B8]/5',
  in_progress: 'text-[#00D4FF] border-[#00D4FF]/30 bg-[#00D4FF]/5',
  completed: 'text-[#6BCB8B] border-[#6BCB8B]/30 bg-[#6BCB8B]/5',
  failed: 'text-red-400 border-red-400/30 bg-red-400/5',
};

function formatDate(ts: number): string {
  return new Date(ts).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

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

/** Extract YouTube thumbnail URL from a video URL (mqdefault = 320×180). */
function youtubeThumbnail(url: string): string | null {
  try {
    const v = new URL(url).searchParams.get('v');
    if (v) return `https://img.youtube.com/vi/${v}/mqdefault.jpg`;
  } catch { /* not a valid URL */ }
  return null;
}

export function RecentRunsSection({ runs, onResume, onDelete }: RecentRunsSectionProps) {
  return (
    <div className="flex flex-col gap-3">
      <h3 className="text-sm font-semibold text-[#A0A0B8]">Pipelines Recentes</h3>
      <div className="flex flex-col gap-2">
        {runs.slice(0, 6).map((run) => {
          const thumb = youtubeThumbnail(run.url);
          return (
            <div
              key={run.id}
              className="flex items-center gap-3 rounded-xl border border-border bg-[#12121C] overflow-hidden"
            >
              {/* Thumbnail */}
              <div className="relative h-16 w-28 shrink-0 bg-[#0D0D15]">
                {thumb ? (
                  <Image
                    src={thumb}
                    alt=""
                    fill
                    className="object-cover"
                    unoptimized
                  />
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-[#5C5C80] text-xs">
                    ▶
                  </div>
                )}
              </div>

              {/* Info */}
              <div className="min-w-0 flex-1 py-2">
                <p className="truncate text-sm text-[#F0F0F8]">{truncateUrl(run.url)}</p>
                <div className="mt-0.5 flex items-center gap-2">
                  <span className="text-xs text-[#5C5C80]">{formatDate(run.startedAt)}</span>
                  {run.mode && run.mode !== '' && (
                    <span className="text-xs text-[#5C5C80]">
                      · {run.mode === 'ai' ? 'Com IA' : 'Cortes'}
                    </span>
                  )}
                </div>
              </div>

              {/* Status + actions */}
              <div className="flex shrink-0 items-center gap-2 pr-3">
                <span
                  className={cn(
                    'rounded-md border px-2 py-0.5 text-xs',
                    STATUS_COLORS[run.status] ?? STATUS_COLORS.pending,
                  )}
                >
                  {STATUS_LABELS[run.status] ?? run.status}
                </span>

                <button
                  type="button"
                  onClick={() => onResume(run.id)}
                  className="rounded-md border border-border bg-[#1A1A26] px-3 py-1.5 text-xs text-[#A0A0B8] transition-colors hover:border-[#A0A0B8]/50 hover:text-[#F0F0F8]"
                >
                  Retomar
                </button>

                <button
                  type="button"
                  onClick={() => onDelete(run.id)}
                  className="rounded-md border border-border bg-[#1A1A26] px-2 py-1.5 text-xs text-[#5C5C80] transition-colors hover:border-red-400/40 hover:text-red-400"
                  aria-label="Remover pipeline"
                >
                  ✕
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
