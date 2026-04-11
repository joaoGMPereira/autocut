'use client';

import { useState } from 'react';
import type { Highlight } from '@/types/pipeline';

interface HighlightCardProps {
  highlight: Highlight;
  videoDurationSec?: number;
  onUpdate: (id: number, adjStart: number, adjEnd: number, isSelected: boolean) => void;
}

function scoreColor(score: number): string {
  if (score >= 0.7) return 'bg-emerald-500 text-white';
  if (score >= 0.4) return 'bg-amber-500 text-white';
  return 'bg-red-500 text-white';
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
    <div className={`rounded-lg border p-3 space-y-2 transition-colors ${isSelected ? 'border-cyan-500/40 bg-cyan-500/5' : 'border-border bg-card/60 opacity-60'}`}>
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={handleToggle}
            className={`w-4 h-4 rounded border shrink-0 flex items-center justify-center transition-colors ${isSelected ? 'border-cyan-400 bg-cyan-400/20' : 'border-zinc-600 bg-transparent'}`}
            aria-label={isSelected ? 'Deselect highlight' : 'Select highlight'}
          >
            {isSelected && <span className="text-cyan-400 text-xs">✓</span>}
          </button>
          <span className={`text-xs px-1.5 py-0.5 rounded font-mono ${scoreColor(highlight.score)}`}>
            {(highlight.score * 100).toFixed(0)}
          </span>
        </div>
        <span className="text-xs text-zinc-500 font-mono">
          {adjStart.toFixed(1)}s – {adjEnd.toFixed(1)}s
        </span>
      </div>

      {highlight.text && (
        <p className="text-xs text-zinc-300 line-clamp-2">{highlight.text}</p>
      )}
      {highlight.reason && (
        <p className="text-xs text-zinc-500 italic">{highlight.reason}</p>
      )}

      {/* Timeline bar */}
      <div className="relative w-full h-3 bg-zinc-800 rounded-full overflow-hidden">
        <div
          className="absolute top-0 h-full bg-cyan-500/60 rounded-full"
          style={{ left: `${startPct}%`, width: `${widthPct}%` }}
        />
      </div>

      {/* Drag handles — range inputs */}
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="text-xs text-zinc-500 block mb-0.5">Start</label>
          <input
            type="range"
            min={0}
            max={videoDurationSec}
            step={0.5}
            value={adjStart}
            onChange={(e) => handleStartChange(parseFloat(e.target.value))}
            className="w-full accent-cyan-400"
          />
        </div>
        <div>
          <label className="text-xs text-zinc-500 block mb-0.5">End</label>
          <input
            type="range"
            min={0}
            max={videoDurationSec}
            step={0.5}
            value={adjEnd}
            onChange={(e) => handleEndChange(parseFloat(e.target.value))}
            className="w-full accent-cyan-400"
          />
        </div>
      </div>
    </div>
  );
}
