'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { PositionGrid } from './PositionGrid';
import { useEffectsStore } from '@/store/effectsStore';

export function LogoTab() {
  const [videoPath, setVideoPath] = useState('');
  const [imagePath, setImagePath] = useState('');
  const [position, setPosition] = useState('bottom_right');
  const [opacity, setOpacity] = useState(0.8);
  const [scale, setScale] = useState(0.3);
  const [outputPath, setOutputPath] = useState('');

  const { applyEffect, overlayJob } = useEffectsStore();
  const isRunning = overlayJob.status === 'running';

  const handleApply = () => {
    if (!videoPath || !imagePath || !outputPath) return;
    void applyEffect(
      'api/effects/overlay',
      { video_path: videoPath, image_path: imagePath, position, opacity, scale, output_path: outputPath },
      'overlayJob',
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="lo-video">Video Path</Label>
        <Input id="lo-video" placeholder="/path/to/video.mp4" value={videoPath} onChange={(e) => setVideoPath(e.target.value)} />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="lo-image">Logo/Watermark Image Path</Label>
        <Input id="lo-image" placeholder="/path/to/logo.png" value={imagePath} onChange={(e) => setImagePath(e.target.value)} />
      </div>

      <div className="flex gap-6 items-start">
        <div className="flex flex-col gap-2">
          <Label>Position</Label>
          <PositionGrid value={position} onChange={setPosition} />
        </div>
        <div className="flex flex-col gap-4 flex-1">
          <div className="flex flex-col gap-2">
            <Label>Opacity: <span className="font-mono text-[#00D4FF]">{opacity.toFixed(2)}</span></Label>
            <Slider
              min={0.1}
              max={1.0}
              step={0.05}
              value={[opacity]}
              onValueChange={([v]) => setOpacity(v)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label>Scale: <span className="font-mono text-[#00D4FF]">{scale.toFixed(2)}x</span></Label>
            <Slider
              min={0.1}
              max={2.0}
              step={0.05}
              value={[scale]}
              onValueChange={([v]) => setScale(v)}
            />
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="lo-output">Output Path</Label>
        <Input id="lo-output" placeholder="/path/to/output.mp4" value={outputPath} onChange={(e) => setOutputPath(e.target.value)} />
      </div>

      <Button
        onClick={handleApply}
        disabled={isRunning || !videoPath || !imagePath || !outputPath}
        className="self-start"
      >
        {isRunning ? 'Applying…' : 'Apply Logo/Watermark'}
      </Button>
    </div>
  );
}
