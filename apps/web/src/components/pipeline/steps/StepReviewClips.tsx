'use client';

import { useEffect, useRef } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { ClipCard } from '../ClipCard';
import { GateActionBar } from '../GateActionBar';
import { createLogger } from '@/lib/logger';

const log = createLogger('StepReviewClips');
const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

export function StepReviewClips() {
  const run = usePipelineStore((s) => s.run);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const submitReviewClips = usePipelineStore((s) => s.submitReviewClips);

  const selectedRef = useRef<Set<number>>(new Set(clips.filter((c) => c.is_selected).map((c) => c.id)));

  useEffect(() => {
    if (run?.id) loadClips(goUrl, run.id);
  }, [run?.id, loadClips]);

  // Sync selectedRef when clips change
  useEffect(() => {
    selectedRef.current = new Set(clips.filter((c) => c.is_selected).map((c) => c.id));
  }, [clips]);

  function handleSelectToggle(id: number, selected: boolean) {
    if (selected) {
      selectedRef.current.add(id);
    } else {
      selectedRef.current.delete(id);
    }
  }

  async function handleSubmit() {
    if (!run) return;
    try {
      await submitReviewClips(goUrl, run.id, { selected_ids: Array.from(selectedRef.current) });
    } catch (err) {
      log.error('[StepReviewClips] submit failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">Review Clips</h2>
        <p className="text-sm text-zinc-400">Select which clips to upload.</p>
      </div>

      {clips.length === 0 ? (
        <p className="text-zinc-500 text-sm">No clips generated yet.</p>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {clips.map((c) => (
            <ClipCard
              key={c.id}
              clip={c}
              variant="select"
              onSelectToggle={handleSelectToggle}
            />
          ))}
        </div>
      )}

      <GateActionBar>
        <button
          type="button"
          onClick={handleSubmit}
          className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-sm font-medium text-white transition-colors"
        >
          Confirm Upload Selection →
        </button>
      </GateActionBar>
    </div>
  );
}
