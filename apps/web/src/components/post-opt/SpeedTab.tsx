'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useEffectsStore } from '@/store/effectsStore';

interface Segment {
  start_sec: number;
  end_sec: number;
  speed: number;
}

const SPEED_PRESETS = [0.5, 1.5, 2, 3] as const;

function emptySegment(): Segment {
  return { start_sec: 0, end_sec: 0, speed: 1 };
}

export function SpeedTab() {
  const [videoPath, setVideoPath] = useState('');
  const [outputPath, setOutputPath] = useState('');
  const [segments, setSegments] = useState<Segment[]>([emptySegment()]);
  const [activeSegIdx, setActiveSegIdx] = useState(0);

  const { applyEffect, speedJob } = useEffectsStore();
  const isRunning = speedJob.status === 'running';

  const updateSegment = (idx: number, patch: Partial<Segment>) => {
    setSegments((prev) => prev.map((s, i) => (i === idx ? { ...s, ...patch } : s)));
  };

  const handleApply = () => {
    if (!videoPath || !outputPath || segments.length === 0) return;
    void applyEffect(
      'api/effects/speed',
      { video_path: videoPath, segments, output_path: outputPath },
      'speedJob',
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-3">
        <div className="flex flex-col gap-1.5 flex-1">
          <Label htmlFor="sp-video">Video Path</Label>
          <Input id="sp-video" placeholder="/path/to/video.mp4" value={videoPath} onChange={(e) => setVideoPath(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5 flex-1">
          <Label htmlFor="sp-output">Output Path</Label>
          <Input id="sp-output" placeholder="/path/to/output.mp4" value={outputPath} onChange={(e) => setOutputPath(e.target.value)} />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <Label>Segments</Label>
          <Button
            variant="outline"
            size="sm"
            onClick={() => { setSegments((p) => [...p, emptySegment()]); setActiveSegIdx(segments.length); }}
          >
            + Add Segment
          </Button>
        </div>

        <div className="flex flex-col gap-2">
          {segments.map((seg, idx) => (
            <div
              key={idx}
              className="flex items-center gap-2 rounded-md border border-border bg-zinc-900/40 px-3 py-2"
              onClick={() => setActiveSegIdx(idx)}
            >
              <span className="text-xs text-zinc-500 w-4">{idx + 1}</span>
              <div className="flex items-center gap-1">
                <Input
                  type="number"
                  className="h-7 w-20 text-xs"
                  placeholder="start"
                  value={seg.start_sec || ''}
                  onChange={(e) => updateSegment(idx, { start_sec: parseFloat(e.target.value) || 0 })}
                />
                <span className="text-zinc-500 text-xs">→</span>
                <Input
                  type="number"
                  className="h-7 w-20 text-xs"
                  placeholder="end"
                  value={seg.end_sec || ''}
                  onChange={(e) => updateSegment(idx, { end_sec: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div className="flex items-center gap-1 ml-2">
                <span className="text-xs text-zinc-500">Speed:</span>
                <Input
                  type="number"
                  className="h-7 w-16 text-xs"
                  step={0.1}
                  value={seg.speed}
                  onChange={(e) => updateSegment(idx, { speed: parseFloat(e.target.value) || 1 })}
                />
                <span className="text-xs text-zinc-500">x</span>
              </div>
              {segments.length > 1 && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 w-6 p-0 ml-auto text-zinc-500 hover:text-red-400"
                  onClick={(e) => { e.stopPropagation(); setSegments((p) => p.filter((_, i) => i !== idx)); }}
                >
                  ×
                </Button>
              )}
            </div>
          ))}
        </div>

        <div className="flex items-center gap-2 mt-1">
          <span className="text-xs text-zinc-500">Presets:</span>
          {SPEED_PRESETS.map((preset) => (
            <Button
              key={preset}
              variant="outline"
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={() => updateSegment(activeSegIdx, { speed: preset })}
            >
              {preset}x
            </Button>
          ))}
        </div>
      </div>

      <Button
        onClick={handleApply}
        disabled={isRunning || !videoPath || !outputPath}
        className="self-start"
      >
        {isRunning ? 'Applying…' : 'Apply Speed Zones'}
      </Button>
    </div>
  );
}
