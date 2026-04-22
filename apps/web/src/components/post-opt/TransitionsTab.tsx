'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { useEffectsStore } from '@/store/effectsStore';

const TRANSITION_TYPES = ['xfade', 'dip_black', 'fade'] as const;
type TransitionType = (typeof TRANSITION_TYPES)[number];

export function TransitionsTab() {
  const [clipPaths, setClipPaths] = useState<string[]>(['', '']);
  const [transitionType, setTransitionType] = useState<TransitionType>('xfade');
  const [duration, setDuration] = useState(0.5);
  const [outputPath, setOutputPath] = useState('');

  const { applyEffect, transitionJob } = useEffectsStore();
  const isRunning = transitionJob.status === 'running';

  const updateClip = (idx: number, val: string) => {
    setClipPaths((prev) => prev.map((p, i) => (i === idx ? val : p)));
  };

  const handleApply = () => {
    const validClips = clipPaths.filter(Boolean);
    if (validClips.length < 2 || !outputPath) return;
    void applyEffect(
      'api/effects/transition',
      { clip_paths: validClips, transition_type: transitionType, transition_duration: duration, output_path: outputPath },
      'transitionJob',
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <Label>Clip Paths</Label>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setClipPaths((p) => [...p, ''])}
          >
            + Add Clip
          </Button>
        </div>
        {clipPaths.map((path, idx) => (
          <div key={idx} className="flex items-center gap-2">
            <span className="text-xs text-zinc-500 w-4">{idx + 1}</span>
            <Input
              placeholder={`/path/to/clip${idx + 1}.mp4`}
              value={path}
              onChange={(e) => updateClip(idx, e.target.value)}
              className="flex-1"
            />
            {clipPaths.length > 2 && (
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0 text-zinc-500 hover:text-red-400"
                onClick={() => setClipPaths((p) => p.filter((_, i) => i !== idx))}
              >
                ×
              </Button>
            )}
          </div>
        ))}
      </div>

      <div className="flex flex-col gap-2">
        <Label>Transition Type</Label>
        <RadioGroup
          value={transitionType}
          onValueChange={(v) => setTransitionType(v as TransitionType)}
          className="flex gap-4"
        >
          {TRANSITION_TYPES.map((t) => (
            <div key={t} className="flex items-center gap-1.5">
              <RadioGroupItem value={t} id={`tr-${t}`} />
              <Label htmlFor={`tr-${t}`} className="cursor-pointer capitalize font-normal">
                {t.replace('_', ' ')}
              </Label>
            </div>
          ))}
        </RadioGroup>
      </div>

      <div className="flex flex-col gap-2">
        <Label>Duration: <span className="font-mono text-brand">{duration.toFixed(1)}s</span></Label>
        <Slider
          min={0.3}
          max={2.0}
          step={0.1}
          value={[duration]}
          onValueChange={([v]) => setDuration(v)}
          className="max-w-xs"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="tr-output">Output Path</Label>
        <Input
          id="tr-output"
          placeholder="/path/to/output.mp4"
          value={outputPath}
          onChange={(e) => setOutputPath(e.target.value)}
        />
      </div>

      <Button
        onClick={handleApply}
        disabled={isRunning || clipPaths.filter(Boolean).length < 2 || !outputPath}
        className="self-start"
      >
        {isRunning ? 'Applying…' : 'Apply Transition'}
      </Button>
    </div>
  );
}
