'use client';

import { MessageCircle } from 'lucide-react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import type { Highlight, HighlightStatus } from '@/store/highlightStore';
import { cn } from '@/lib/utils';

function formatSec(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

function confidenceColor(c: number): string {
  if (c < 0.5) return 'bg-destructive';
  if (c < 0.7) return 'bg-warning';
  return 'bg-success';
}

function confidenceTextColor(c: number): string {
  if (c < 0.5) return 'text-destructive';
  if (c < 0.7) return 'text-warning';
  return 'text-success';
}

interface HighlightRowProps {
  highlight: Highlight;
}

function HighlightRow({ highlight }: HighlightRowProps) {
  const { startSec, endSec, confidence, strategies, scores } = highlight;
  const scoreText = Object.entries(scores)
    .map(([k, v]) => `${k}: ${v.toFixed(2)}`)
    .join(' | ');

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-surface/50 px-3 py-2.5">
      {/* Timestamp + confidence value */}
      <div className="flex items-center justify-between">
        <span className="font-mono text-xs text-prose">
          {formatSec(startSec)} – {formatSec(endSec)}
        </span>
        <span className={cn('font-mono text-xs font-semibold', confidenceTextColor(confidence))}>
          {(confidence * 100).toFixed(0)}%
        </span>
      </div>

      {/* Confidence bar */}
      <div className="h-1.5 w-full rounded-full bg-surface overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all', confidenceColor(confidence))}
          style={{ width: `${Math.min(confidence * 100, 100)}%` }}
        />
      </div>

      {/* Strategy badges with scores tooltip */}
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex flex-wrap gap-1">
              {strategies.map((s) => (
                <Badge
                  key={s}
                  variant="secondary"
                  className="text-[10px] px-1.5 py-0 h-4 cursor-default flex items-center gap-0.5"
                >
                  {s === 'chat' && <MessageCircle className="h-2.5 w-2.5" />}
                  {s}
                </Badge>
              ))}
            </div>
          </TooltipTrigger>
          <TooltipContent side="bottom">
            <span className="font-mono text-xs">{scoreText}</span>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}

interface HighlightListProps {
  highlights: Highlight[];
  threshold: number;
  status: HighlightStatus;
}

export function HighlightList({ highlights, threshold, status }: HighlightListProps) {
  const visible = highlights.filter((h) => h.confidence >= threshold);

  if (status === 'idle') {
    return (
      <div className="flex items-center justify-center h-40 rounded-xl border border-border text-caption text-sm">
        Execute a detecção para ver os highlights
      </div>
    );
  }

  if (status === 'running' && visible.length === 0) {
    return (
      <div className="flex items-center justify-center h-40 rounded-xl border border-border text-subtle text-sm animate-pulse">
        Detectando highlights…
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div className="flex items-center justify-center h-40 rounded-xl border border-destructive/40 text-destructive text-sm">
        Erro na detecção. Verifique os logs.
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {/* Summary bar */}
      <div className="flex items-center justify-between px-1">
        <span className="text-xs text-subtle">
          {visible.length} highlight{visible.length !== 1 ? 's' : ''} detectado{visible.length !== 1 ? 's' : ''}
          {visible.length < highlights.length && (
            <span className="text-caption ml-1">
              ({highlights.length - visible.length} filtrado{highlights.length - visible.length !== 1 ? 's' : ''})
            </span>
          )}
        </span>
        {status === 'running' && (
          <span className="text-xs text-info animate-pulse">Processando…</span>
        )}
        {status === 'done' && (
          <span className="text-xs text-success">Concluído</span>
        )}
      </div>

      <ScrollArea className="h-[420px] pr-2">
        <div className="flex flex-col gap-2">
          {visible.map((h, i) => (
            <HighlightRow key={`${h.startSec}-${i}`} highlight={h} />
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}
