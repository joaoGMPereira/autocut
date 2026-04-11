'use client';

import { useEffect, useRef } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { HighlightCard } from '../HighlightCard';
import { GateActionBar } from '../GateActionBar';
import type { HighlightUpdate } from '@/types/pipeline';
import { createLogger } from '@/lib/logger';

const log = createLogger('StepReviewHighlights');
const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

export function StepReviewHighlights() {
  const run = usePipelineStore((s) => s.run);
  const highlights = usePipelineStore((s) => s.highlights);
  const loadHighlights = usePipelineStore((s) => s.loadHighlights);
  const submitReviewHighlights = usePipelineStore((s) => s.submitReviewHighlights);

  // Local edits accumulator (keyed by id)
  const editsRef = useRef<Map<number, HighlightUpdate>>(new Map());

  useEffect(() => {
    if (run?.id) loadHighlights(goUrl, run.id);
  }, [run?.id, loadHighlights]);

  function handleUpdate(id: number, adjStart: number, adjEnd: number, isSelected: boolean) {
    editsRef.current.set(id, { id, adj_start_sec: adjStart, adj_end_sec: adjEnd, is_selected: isSelected });
  }

  async function handleSubmit() {
    if (!run) return;
    const updates = highlights.map((h) => {
      const edit = editsRef.current.get(h.id);
      return edit ?? { id: h.id, adj_start_sec: h.adj_start_sec, adj_end_sec: h.adj_end_sec, is_selected: h.is_selected };
    });
    try {
      await submitReviewHighlights(goUrl, run.id, { highlights: updates });
    } catch (err) {
      log.error('[StepReviewHighlights] submit failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  }

  const selectedCount = highlights.filter((h) => {
    const edit = editsRef.current.get(h.id);
    return edit ? edit.is_selected : h.is_selected;
  }).length;

  return (
    <div className="space-y-4 max-w-2xl">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">Review Highlights</h2>
        <p className="text-sm text-zinc-400">
          {highlights.length} highlight{highlights.length !== 1 ? 's' : ''} detected.
          {' '}{selectedCount} selected for clipping.
        </p>
      </div>

      {highlights.length === 0 ? (
        <p className="text-zinc-500 text-sm">No highlights found. You may still proceed.</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {highlights.map((h) => (
            <HighlightCard
              key={h.id}
              highlight={h}
              onUpdate={handleUpdate}
            />
          ))}
        </div>
      )}

      <GateActionBar>
        <span className="text-xs text-zinc-400 mr-auto">{selectedCount} highlight{selectedCount !== 1 ? 's' : ''} selected</span>
        <button
          type="button"
          onClick={handleSubmit}
          className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-sm font-medium text-white transition-colors"
        >
          Generate Clips →
        </button>
      </GateActionBar>
    </div>
  );
}
