'use client';

import { useState } from 'react';
import { Badge } from '@/components/ui/badge';
import type { Highlight } from '@/types/pipeline';

interface HighlightCardProps {
  highlight: Highlight;
  videoDurationSec?: number;
  onUpdate: (id: number, adjStart: number, adjEnd: number, isSelected: boolean) => void;
}

function scoreVariant(score: number): 'success' | 'warning' | 'destructive' {
  if (score >= 0.7) return 'success';
  if (score >= 0.4) return 'warning';
  return 'destructive';
}

export function HighlightCard({ highlight, videoDurationSec = 600, onUpdate }: HighlightCardProps) {
  const [adjStart, setAdjStart] = useState(highlight.adj_start_sec);
  const [adjEnd, setAdjEnd] = useState(highlight.adj_end_sec);
  const [isSelected, setIsSelected] = useState(highlight.is_selected);

  function handleStartChange(v: number) {
    const clamped = Math.max(0, Math.min(v, adjEnd - 1));
    setAdjStart(clamped);
    onUpdate(highlight.id, clamped, adjEnd, isSelected);
  }

  function handleEndChange(v: number) {
    const clamped = Math.max(adjStart + 1, Math.min(v, videoDurationSec));
    setAdjEnd(clamped);
    onUpdate(highlight.id, adjStart, clamped, isSelected);
  }

  function handleToggle() {
    const next = !isSelected;
    setIsSelected(next);
    onUpdate(highlight.id, adjStart, adjEnd, next);
  }

  const startPct = (adjStart / videoDurationSec) * 100;
  const widthPct = ((adjEnd - adjStart) / videoDurationSec) * 100;

  return (
    <div className={`rounded-lg border p-3 space-y-2 transition-colors ${isSelected ? 'border-brand/40 bg-brand/5' : 'border-border bg-card/60 opacity-60'}`}>
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={handleToggle}
            className={`w-4 h-4 rounded border shrink-0 flex items-center justify-center transition-colors ${isSelected ? 'border-brand bg-brand/20' : 'border-caption bg-transparent'}`}
            aria-label={isSelected ? 'Deselect highlight' : 'Select highlight'}
          >
            {isSelected && <span className="text-brand text-xs">✓</span>}
          </button>
          <Badge variant={scoreVariant(highlight.score)} className="font-mono">
            {(highlight.score * 100).toFixed(0)}
          </Badge>
        </div>
        <span className="text-xs text-subtle font-mono">
          {adjStart.toFixed(1)}s – {adjEnd.toFixed(1)}s
        </span>
      </div>

      {highlight.text && (
        <p className="text-xs text-prose line-clamp-2">{highlight.text}</p>
      )}
      {highlight.reason && (
        <p className="text-xs text-subtle italic">{highlight.reason}</p>
      )}

      {/* Timeline bar */}
      <div className="relative w-full h-3 bg-surface rounded-full overflow-hidden">
        <div
          className="absolute top-0 h-full bg-brand/60 rounded-full"
          style={{ left: `${startPct}%`, width: `${widthPct}%` }}
        />
      </div>

      {/* Drag handles — range inputs */}
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="text-xs text-subtle block mb-0.5">Start</label>
          <input
            type="range"
            min={0}
            max={videoDurationSec}
            step={0.5}
            value={adjStart}
            onChange={(e) => handleStartChange(parseFloat(e.target.value))}
            className="w-full accent-brand"
          />
        </div>
        <div>
          <label className="text-xs text-subtle block mb-0.5">End</label>
          <input
            type="range"
            min={0}
            max={videoDurationSec}
            step={0.5}
            value={adjEnd}
            onChange={(e) => handleEndChange(parseFloat(e.target.value))}
            className="w-full accent-brand"
          />
        </div>
      </div>
    </div>
  );
}
