'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { PositionGrid } from './PositionGrid';
import { useEffectsStore } from '@/store/effectsStore';

export function TextOverlayTab() {
  const [videoPath, setVideoPath] = useState('');
  const [text, setText] = useState('');
  const [fontName, setFontName] = useState('Arial');
  const [fontSize, setFontSize] = useState(48);
  const [color, setColor] = useState('#FFFFFF');
  const [position, setPosition] = useState('mid_center');
  const [startSec, setStartSec] = useState(0);
  const [endSec, setEndSec] = useState(5);
  const [outputPath, setOutputPath] = useState('');

  const { applyEffect, textJob } = useEffectsStore();
  const isRunning = textJob.status === 'running';

  const handleApply = () => {
    if (!videoPath || !text || !outputPath) return;
    void applyEffect(
      'api/effects/text',
      {
        video_path: videoPath,
        text,
        font_name: fontName,
        font_size: fontSize,
        color,
        position,
        start_sec: startSec,
        end_sec: endSec,
        output_path: outputPath,
      },
      'textJob',
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-3">
        <div className="flex flex-col gap-1.5 flex-1">
          <Label htmlFor="tx-video">Video Path</Label>
          <Input id="tx-video" placeholder="/path/to/video.mp4" value={videoPath} onChange={(e) => setVideoPath(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5 flex-1">
          <Label htmlFor="tx-output">Output Path</Label>
          <Input id="tx-output" placeholder="/path/to/output.mp4" value={outputPath} onChange={(e) => setOutputPath(e.target.value)} />
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="tx-text">Text</Label>
        <Input id="tx-text" placeholder="Overlay text…" value={text} onChange={(e) => setText(e.target.value)} />
      </div>

      <div className="flex gap-3">
        <div className="flex flex-col gap-1.5 flex-1">
          <Label htmlFor="tx-font">Font Name</Label>
          <Input id="tx-font" placeholder="Arial" value={fontName} onChange={(e) => setFontName(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5 w-20">
          <Label htmlFor="tx-color">Color</Label>
          <input
            id="tx-color"
            type="color"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            className="h-9 w-full rounded-md border border-input bg-transparent cursor-pointer"
          />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <Label>Font Size: <span className="font-mono text-[#00D4FF]">{fontSize}px</span></Label>
        <Slider min={12} max={120} step={1} value={[fontSize]} onValueChange={([v]) => setFontSize(v)} className="max-w-xs" />
      </div>

      <div className="flex gap-6">
        <div className="flex flex-col gap-2">
          <Label>Position</Label>
          <PositionGrid value={position} onChange={setPosition} />
        </div>
        <div className="flex flex-col gap-3 justify-center">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tx-start">Start (sec)</Label>
            <Input
              id="tx-start"
              type="number"
              className="w-24"
              value={startSec}
              onChange={(e) => setStartSec(parseFloat(e.target.value) || 0)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tx-end">End (sec)</Label>
            <Input
              id="tx-end"
              type="number"
              className="w-24"
              value={endSec}
              onChange={(e) => setEndSec(parseFloat(e.target.value) || 0)}
            />
          </div>
        </div>
      </div>

      <Button
        onClick={handleApply}
        disabled={isRunning || !videoPath || !text || !outputPath}
        className="self-start"
      >
        {isRunning ? 'Applying…' : 'Apply Text Overlay'}
      </Button>
    </div>
  );
}
