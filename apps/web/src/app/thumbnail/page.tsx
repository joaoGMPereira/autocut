'use client';

import { Anton } from 'next/font/google';
import { useEffect, useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';
import { useThumbnailStore } from '@/store/thumbnailStore';
import { Button } from '@/components/ui/button';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { BackgroundsGallery } from '@/components/thumbnail/BackgroundsGallery';
import { Loader2 } from 'lucide-react';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

type Strategy = 'branded' | 'centered' | 'template' | 'ai';

export default function ThumbnailPage() {
  const { goUrl } = useAppStore();
  const { channels, loading: channelsLoading, fetchChannels } = useChannelStore();
  const { selectedBgId, fetchBackgrounds } = useThumbnailStore();

  const [videoPath, setVideoPath] = useState('');
  const [strategy, setStrategy] = useState<Strategy>('branded');
  const [text, setText] = useState('');
  const [fontColor, setFontColor] = useState('#FFFFFF');
  const [fontSize, setFontSize] = useState(72);
  const [output, setOutput] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ outputPath: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  // template-specific state
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);

  // ai-specific state
  const [customPrompt, setCustomPrompt] = useState('');

  const derivedOutput = videoPath ? videoPath.replace(/\.[^.]+$/, '_thumb.jpg') : '';
  const effectiveOutput = output || derivedOutput;

  // Load channels when template strategy is selected
  useEffect(() => {
    if (strategy === 'template' && channels.length === 0) {
      void fetchChannels(goUrl);
    }
  }, [strategy, channels.length, fetchChannels, goUrl]);

  // Load backgrounds when a channel is selected
  useEffect(() => {
    if (strategy === 'template' && selectedChannelId !== null) {
      void fetchBackgrounds(selectedChannelId);
    }
  }, [strategy, selectedChannelId, fetchBackgrounds]);

  const handleGenerate = async () => {
    if (!videoPath.trim()) return;
    setLoading(true);
    setResult(null);
    setError(null);
    try {
      const body: Record<string, unknown> = {
        video_path: videoPath.trim(),
        strategy,
        text: text.trim(),
        font_color: fontColor,
        font_size: fontSize,
        output: effectiveOutput,
      };
      if (strategy === 'template' && selectedBgId !== null) {
        body.background_id = selectedBgId;
      }
      if (strategy === 'ai' && customPrompt.trim() !== '') {
        body.custom_prompt = customPrompt.trim();
      }
      const res = await fetch(`${goUrl}/api/thumbnail`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
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

  const isGenerateDisabled =
    loading ||
    !videoPath.trim() ||
    (strategy === 'template' && selectedBgId === null);

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      {/* Header */}
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        THUMBNAIL
      </h1>

      {/* Form card */}
      <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-5">
        <h2 className="text-sm font-semibold">Generate Thumbnail</h2>

        {/* Strategy selector */}
        <div className="flex flex-col gap-2">
          <span className="text-xs text-[#5C5C80]">Strategy</span>
          <RadioGroup
            value={strategy}
            onValueChange={(v) => setStrategy(v as Strategy)}
            className="flex flex-row gap-5"
          >
            {(['branded', 'centered', 'template', 'ai'] as const).map((s) => (
              <div key={s} className="flex items-center gap-2">
                <RadioGroupItem value={s} id={`strategy-${s}`} />
                <Label htmlFor={`strategy-${s}`} className="capitalize cursor-pointer">
                  {s === 'ai' ? 'AI-Generated' : s}
                </Label>
              </div>
            ))}
          </RadioGroup>
        </div>

        {/* AI-specific: custom Ollama prompt + info note */}
        {strategy === 'ai' && (
          <div className="flex flex-col gap-3 rounded-lg border border-border bg-[#12121C] p-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-xs text-[#5C5C80]">Custom Ollama Prompt</label>
              <textarea
                value={customPrompt}
                onChange={(e) => setCustomPrompt(e.target.value)}
                placeholder="Optional: customize the AI prompt for generating thumbnail text..."
                rows={3}
                className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 resize-none"
              />
            </div>
            <p className="text-xs text-[#5C5C80]">
              Uses Ollama (llava or llama3) to generate CTR-optimized thumbnail text from a video
              frame. Falls back to a generic title if Ollama is unavailable.
            </p>
          </div>
        )}

        {/* Template-specific: channel + backgrounds gallery */}
        {strategy === 'template' && (
          <div className="flex flex-col gap-4 rounded-lg border border-border bg-[#12121C] p-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-xs text-[#5C5C80]">Channel</label>
              {channelsLoading ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Loading channels…
                </div>
              ) : (
                <Select
                  value={selectedChannelId !== null ? String(selectedChannelId) : ''}
                  onValueChange={(v) => setSelectedChannelId(Number(v))}
                >
                  <SelectTrigger className="w-full max-w-xs">
                    <SelectValue placeholder="Select a channel" />
                  </SelectTrigger>
                  <SelectContent>
                    {channels.map((ch) => (
                      <SelectItem key={ch.ID} value={String(ch.ID)}>
                        {ch.Name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>

            {selectedChannelId !== null && (
              <div className="flex flex-col gap-2">
                <span className="text-xs text-[#5C5C80]">Background Image</span>
                <BackgroundsGallery channelId={selectedChannelId} />
                {selectedBgId === null && (
                  <p className="text-xs text-amber-400/80">
                    Select a background to enable generation.
                  </p>
                )}
              </div>
            )}
          </div>
        )}

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
          disabled={isGenerateDisabled}
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
