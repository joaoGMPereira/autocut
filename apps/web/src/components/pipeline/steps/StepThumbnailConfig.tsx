'use client';

import { useEffect, useState } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { ThumbnailStylePicker } from '../ThumbnailStylePicker';
import { GateActionBar } from '../GateActionBar';
import type { ClipThumbnailOverride } from '@/types/pipeline';
import { createLogger } from '@/lib/logger';

const log = createLogger('StepThumbnailConfig');
const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

export function StepThumbnailConfig() {
  const run = usePipelineStore((s) => s.run);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const submitThumbnailConfig = usePipelineStore((s) => s.submitThumbnailConfig);

  const [defaultStyle, setDefaultStyle] = useState('default');
  const [overrides, setOverrides] = useState<Map<number, string>>(new Map());

  useEffect(() => {
    if (run?.id) loadClips(goUrl, run.id);
  }, [run?.id, loadClips]);

  function setClipStyle(clipId: number, style: string) {
    setOverrides((prev) => {
      const next = new Map(prev);
      next.set(clipId, style);
      return next;
    });
  }

  async function handleSubmit() {
    if (!run) return;
    const clipOverrides: ClipThumbnailOverride[] = Array.from(overrides.entries()).map(
      ([clip_id, style]) => ({ clip_id, style }),
    );
    try {
      await submitThumbnailConfig(goUrl, run.id, { default_style: defaultStyle, clip_overrides: clipOverrides });
    } catch (err) {
      log.error('[StepThumbnailConfig] submit failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  }

  return (
    <div className="space-y-6 max-w-lg">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">Thumbnail Style</h2>
        <p className="text-sm text-zinc-400">Choose a default style. Override per clip below.</p>
      </div>

      <div>
        <p className="text-sm text-zinc-300 mb-2">Default style</p>
        <ThumbnailStylePicker value={defaultStyle} onChange={setDefaultStyle} />
      </div>

      {clips.length > 0 && (
        <div>
          <p className="text-sm text-zinc-300 mb-2">Per-clip overrides ({clips.length} clips)</p>
          <div className="space-y-3 max-h-64 overflow-y-auto pr-1">
            {clips.map((c) => (
              <div key={c.id} className="rounded-lg border border-border bg-card p-3 space-y-2">
                <p className="text-xs text-zinc-400 font-mono">Clip {c.id}</p>
                <ThumbnailStylePicker
                  value={overrides.get(c.id) ?? defaultStyle}
                  onChange={(s) => setClipStyle(c.id, s)}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      <GateActionBar>
        <button
          type="button"
          onClick={handleSubmit}
          className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-sm font-medium text-white transition-colors"
        >
          Review Metadata →
        </button>
      </GateActionBar>
    </div>
  );
}
