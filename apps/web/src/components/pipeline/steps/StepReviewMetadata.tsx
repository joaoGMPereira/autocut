'use client';

import { useEffect, useRef } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { ClipCard } from '../ClipCard';
import { GateActionBar } from '../GateActionBar';
import type { ClipMetadataUpdate } from '@/types/pipeline';
import { createLogger } from '@/lib/logger';

const log = createLogger('StepReviewMetadata');
const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

export function StepReviewMetadata() {
  const run = usePipelineStore((s) => s.run);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const submitReviewMetadata = usePipelineStore((s) => s.submitReviewMetadata);

  const editsRef = useRef<Map<number, ClipMetadataUpdate>>(new Map());

  useEffect(() => {
    if (run?.id) loadClips(goUrl, run.id);
  }, [run?.id, loadClips]);

  function handleMetadataChange(id: number, title: string, description: string, tags: string) {
    editsRef.current.set(id, { id, title, description, tags });
  }

  async function handleSubmit() {
    if (!run) return;
    const clipUpdates: ClipMetadataUpdate[] = clips.map((c) => {
      const edit = editsRef.current.get(c.id);
      return edit ?? { id: c.id, title: c.title, description: c.description, tags: c.tags };
    });
    try {
      await submitReviewMetadata(goUrl, run.id, { clips: clipUpdates });
    } catch (err) {
      log.error('[StepReviewMetadata] submit failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">Review Metadata</h2>
        <p className="text-sm text-zinc-400">Edit title, description, and tags for each clip.</p>
      </div>

      {clips.length === 0 ? (
        <p className="text-zinc-500 text-sm">No clips to review.</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {clips.map((c) => (
            <ClipCard
              key={c.id}
              clip={c}
              variant="metadata"
              onMetadataChange={handleMetadataChange}
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
          Review Clips →
        </button>
      </GateActionBar>
    </div>
  );
}
