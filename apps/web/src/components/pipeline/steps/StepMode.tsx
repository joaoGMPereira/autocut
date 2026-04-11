'use client';

import { useState } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { GateActionBar } from '../GateActionBar';
import type { WorkflowMode, ModeConfig } from '@/types/pipeline';
import { createLogger } from '@/lib/logger';

const log = createLogger('StepMode');
const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

export function StepMode() {
  const run = usePipelineStore((s) => s.run);
  const advance = usePipelineStore((s) => s.advance);

  const [mode, setMode] = useState<WorkflowMode>('ai');
  const [minDuration, setMinDuration] = useState(30);
  const [maxDuration, setMaxDuration] = useState(120);
  const [sensitivity, setSensitivity] = useState(70);
  const [segmentSecs, setSegmentSecs] = useState(60);
  const [loading, setLoading] = useState(false);

  async function handleStart() {
    if (!run) return;
    setLoading(true);
    try {
      const modeConfig: ModeConfig = {
        mode,
        min_duration_secs: minDuration,
        max_duration_secs: maxDuration,
        ...(mode === 'ai' ? { sensitivity_pct: sensitivity } : { segment_secs: segmentSecs }),
      };
      await advance(goUrl, run.id, { mode_config: modeConfig });
    } catch (err) {
      log.error('[StepMode] advance failed', { error: err instanceof Error ? err.message : 'Unknown' });
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6 max-w-xl">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">Workflow Mode</h2>
        <p className="text-sm text-zinc-400">Choose how to process the video.</p>
      </div>

      {/* Mode cards */}
      <div className="grid grid-cols-2 gap-3">
        {(['ai', 'longform'] as WorkflowMode[]).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => setMode(m)}
            className={`rounded-xl border p-4 text-left transition-colors ${
              mode === m ? 'border-cyan-500 bg-cyan-500/10' : 'border-border bg-card hover:border-zinc-500'
            }`}
          >
            <div className="text-sm font-semibold text-foreground capitalize mb-1">
              {m === 'ai' ? 'AI Highlights' : 'Long Form'}
            </div>
            <div className="text-xs text-zinc-400">
              {m === 'ai'
                ? 'Automatically detect highlights using AI transcription and topic detection.'
                : 'Split video into fixed-duration segments for manual review.'}
            </div>
          </button>
        ))}
      </div>

      {/* Config sliders */}
      <div className="space-y-4">
        <div>
          <div className="flex justify-between text-xs text-zinc-400 mb-1">
            <span>Min duration</span><span>{minDuration}s</span>
          </div>
          <input type="range" min={10} max={300} step={5} value={minDuration}
            onChange={(e) => setMinDuration(Number(e.target.value))}
            className="w-full accent-cyan-400" />
        </div>
        <div>
          <div className="flex justify-between text-xs text-zinc-400 mb-1">
            <span>Max duration</span><span>{maxDuration}s</span>
          </div>
          <input type="range" min={30} max={600} step={10} value={maxDuration}
            onChange={(e) => setMaxDuration(Number(e.target.value))}
            className="w-full accent-cyan-400" />
        </div>
        {mode === 'ai' && (
          <div>
            <div className="flex justify-between text-xs text-zinc-400 mb-1">
              <span>AI Sensitivity</span><span>{sensitivity}%</span>
            </div>
            <input type="range" min={50} max={90} step={5} value={sensitivity}
              onChange={(e) => setSensitivity(Number(e.target.value))}
              className="w-full accent-cyan-400" />
          </div>
        )}
        {mode === 'longform' && (
          <div>
            <div className="flex justify-between text-xs text-zinc-400 mb-1">
              <span>Segment length</span><span>{segmentSecs}s</span>
            </div>
            <input type="range" min={30} max={600} step={30} value={segmentSecs}
              onChange={(e) => setSegmentSecs(Number(e.target.value))}
              className="w-full accent-cyan-400" />
          </div>
        )}
      </div>

      <GateActionBar>
        <button
          type="button"
          onClick={handleStart}
          disabled={loading}
          className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed text-sm font-medium text-white transition-colors"
        >
          {loading ? 'Starting…' : 'Start Processing →'}
        </button>
      </GateActionBar>
    </div>
  );
}
