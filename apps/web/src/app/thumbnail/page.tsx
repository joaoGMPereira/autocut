'use client';

import { Anton } from 'next/font/google';
import { useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { Button } from '@/components/ui/button';
import { Loader2 } from 'lucide-react';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

export default function ThumbnailPage() {
  const { goUrl } = useAppStore();

  const [videoPath, setVideoPath] = useState('');
  const [template, setTemplate] = useState<'branded' | 'centered'>('branded');
  const [text, setText] = useState('');
  const [fontColor, setFontColor] = useState('#FFFFFF');
  const [fontSize, setFontSize] = useState(72);
  const [output, setOutput] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ outputPath: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const derivedOutput = videoPath ? videoPath.replace(/\.[^.]+$/, '_thumb.jpg') : '';
  const effectiveOutput = output || derivedOutput;

  const handleGenerate = async () => {
    if (!videoPath.trim()) return;
    setLoading(true);
    setResult(null);
    setError(null);
    try {
      const res = await fetch(`${goUrl}/api/thumbnail`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          video_path: videoPath.trim(),
          template,
          text: text.trim(),
          font_color: fontColor,
          font_size: fontSize,
          output: effectiveOutput,
        }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as { output_path: string };
      setResult({ outputPath: data.output_path });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Generation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      {/* Header */}
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        THUMBNAIL
      </h1>

      {/* Form card */}
      <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-5">
        <h2 className="text-sm font-semibold">Generate Thumbnail</h2>

        <div className="grid grid-cols-2 gap-4">
          <div className="col-span-2">
            <label className="text-xs text-[#5C5C80] mb-1 block">Video Path</label>
            <input
              type="text"
              value={videoPath}
              onChange={(e) => {
                setVideoPath(e.target.value);
                setOutput('');
              }}
              placeholder="/path/to/video.mp4"
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>

          <div>
            <label className="text-xs text-[#5C5C80] mb-1 block">Template</label>
            <select
              value={template}
              onChange={(e) => setTemplate(e.target.value as 'branded' | 'centered')}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 appearance-none"
            >
              <option value="branded">Branded</option>
              <option value="centered">Centered</option>
            </select>
          </div>

          <div>
            <label className="text-xs text-[#5C5C80] mb-1 block">Overlay Text</label>
            <input
              type="text"
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Optional text overlay"
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>

          <div>
            <label className="text-xs text-[#5C5C80] mb-1 block">Font Color</label>
            <input
              type="color"
              value={fontColor}
              onChange={(e) => setFontColor(e.target.value)}
              className="h-9 w-16 rounded cursor-pointer bg-[#1A1A26] border border-border px-1"
            />
          </div>

          <div>
            <label className="text-xs text-[#5C5C80] mb-1 block">Font Size</label>
            <input
              type="number"
              value={fontSize}
              min={48}
              max={120}
              onChange={(e) => setFontSize(Number(e.target.value))}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>

          <div className="col-span-2">
            <label className="text-xs text-[#5C5C80] mb-1 block">
              Output Path{' '}
              <span className="text-[#5C5C80]/60">(auto-derived if empty)</span>
            </label>
            <input
              type="text"
              value={output || derivedOutput}
              onChange={(e) => setOutput(e.target.value)}
              placeholder={derivedOutput || '/path/to/output_thumb.jpg'}
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
        </div>

        <Button
          onClick={() => void handleGenerate()}
          disabled={loading || !videoPath.trim()}
          className="w-fit"
        >
          {loading ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Generating…
            </>
          ) : (
            'Generate'
          )}
        </Button>
      </div>

      {/* Success state */}
      {result && (
        <div className="rounded-xl bg-emerald-500/10 border border-emerald-500/20 p-5">
          <p className="text-sm text-emerald-400 font-medium">Thumbnail generated</p>
          <p className="text-xs text-emerald-400/80 mt-1 font-mono">{result.outputPath}</p>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="rounded-xl bg-destructive/10 border border-destructive/20 p-5">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}
    </div>
  );
}
