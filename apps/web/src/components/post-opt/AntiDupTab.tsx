'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useEffectsStore } from '@/store/effectsStore';

export function AntiDupTab() {
  const [videoPath, setVideoPath] = useState('');
  const [captionText, setCaptionText] = useState('');
  const [outputPath, setOutputPath] = useState('');

  const { applyEffect, antiDupJob } = useEffectsStore();
  const isRunning = antiDupJob.status === 'running';

  const handleApply = () => {
    if (!videoPath || !outputPath) return;
    void applyEffect(
      'api/effects/anti-dup',
      { video_path: videoPath, caption_text: captionText, output_path: outputPath },
      'antiDupJob',
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ad-video">Video Path</Label>
        <Input
          id="ad-video"
          placeholder="/path/to/video.mp4"
          value={videoPath}
          onChange={(e) => setVideoPath(e.target.value)}
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ad-caption">Caption Text</Label>
        <textarea
          id="ad-caption"
          placeholder="Enter caption text for duplicate detection…"
          value={captionText}
          onChange={(e) => setCaptionText(e.target.value)}
          rows={3}
          className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-none"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ad-output">Output Path</Label>
        <Input
          id="ad-output"
          placeholder="/path/to/output.mp4"
          value={outputPath}
          onChange={(e) => setOutputPath(e.target.value)}
        />
      </div>

      <Button
        onClick={handleApply}
        disabled={isRunning || !videoPath || !outputPath}
        className="self-start"
      >
        {isRunning ? 'Applying…' : 'Apply Anti-Duplicate'}
      </Button>
    </div>
  );
}
