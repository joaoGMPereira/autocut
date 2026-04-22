'use client';

import { useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { useProcessorStore } from '@/store/processorStore';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { createLogger } from '@/lib/logger';
import type { JobStatus } from '@/store/processorStore';
import { ShortsResultList } from './ShortsResultList';

const log = createLogger('ShortsPage');

const PRESETS = [
  { label: '9:16 (1080×1920)', width: 1080, height: 1920 },
  { label: '1:1 (1080×1080)', width: 1080, height: 1080 },
  { label: 'Custom', width: 0, height: 0 },
] as const;

function StatusBadge({ status }: { status: JobStatus }) {
  if (status === 'running') {
    return (
      <span className="bg-blue-500/10 text-blue-400 text-xs px-2 py-0.5 rounded-md">
        running
      </span>
    );
  }
  if (status === 'done') {
    return (
      <span className="bg-emerald-500/10 text-emerald-400 text-xs px-2 py-0.5 rounded-md">
        done
      </span>
    );
  }
  if (status === 'error') {
    return (
      <span className="bg-destructive/10 text-destructive text-xs px-2 py-0.5 rounded-md">
        error
      </span>
    );
  }
  return (
    <span className="bg-surface text-subtle text-xs px-2 py-0.5 rounded-md">
      idle
    </span>
  );
}

export function ShortsForm() {
  const { goUrl } = useAppStore();
  const { shortsJobs, shortsConfig, setShortsConfig, startShorts } = useProcessorStore();

  const [input, setInput] = useState('');
  const [output, setOutput] = useState('');
  const [preset, setPreset] = useState(0);
  const [customWidth, setCustomWidth] = useState(1080);
  const [customHeight, setCustomHeight] = useState(1920);
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const isCustom = PRESETS[preset].label === 'Custom';
  const width = isCustom ? customWidth : PRESETS[preset].width;
  const height = isCustom ? customHeight : PRESETS[preset].height;

  const activeJob = activeJobId ? shortsJobs[activeJobId] : null;
  const isRunning = activeJob?.status === 'running';

  const handleCreate = async () => {
    setError(null);
    log.info('starting shorts creation', { input, output, width, height, shortsConfig });
    try {
      const id = await startShorts(goUrl, {
        input,
        output,
        width,
        height,
        add_blur_background: shortsConfig.addBlurBackground,
        add_captions: shortsConfig.addCaptions,
        transcript_path: shortsConfig.addCaptions ? shortsConfig.transcriptPath : undefined,
        language: shortsConfig.language,
        generate_thumbnails: shortsConfig.generateThumbnails,
        min_duration: shortsConfig.minDuration > 0 ? shortsConfig.minDuration : undefined,
        max_duration: shortsConfig.maxDuration > 0 ? shortsConfig.maxDuration : undefined,
      });
      setActiveJobId(id);
      log.info('shorts job started', { jobId: id });
    } catch (err) {
      log.error('failed to start shorts', { err });
      setError(err instanceof Error ? err.message : 'Failed to start');
    }
  };

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className="font-display text-[32px] font-bold tracking-[-0.02em] text-heading">
        Shorts
      </h1>

      <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-4">
        {/* Input path */}
        <div className="flex flex-col gap-1">
          <label className="text-xs text-subtle mb-1 block">Input Path</label>
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="/path/to/input.mp4"
            className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60"
          />
        </div>

        {/* Output path */}
        <div className="flex flex-col gap-1">
          <label className="text-xs text-subtle mb-1 block">Output Path</label>
          <input
            type="text"
            value={output}
            onChange={(e) => setOutput(e.target.value)}
            placeholder="/path/to/output/"
            className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60"
          />
        </div>

        {/* Aspect ratio preset */}
        <div className="flex flex-col gap-1">
          <label className="text-xs text-subtle mb-1 block">Aspect Ratio</label>
          <select
            value={preset}
            onChange={(e) => setPreset(Number(e.target.value))}
            className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60"
          >
            {PRESETS.map((p, i) => (
              <option key={p.label} value={i}>
                {p.label}
              </option>
            ))}
          </select>
        </div>

        {/* Custom dimensions */}
        {isCustom && (
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1">
              <label className="text-xs text-subtle mb-1 block">Width (px)</label>
              <input
                type="number"
                value={customWidth}
                onChange={(e) => setCustomWidth(Number(e.target.value))}
                min={1}
                className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60"
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-subtle mb-1 block">Height (px)</label>
              <input
                type="number"
                value={customHeight}
                onChange={(e) => setCustomHeight(Number(e.target.value))}
                min={1}
                className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60"
              />
            </div>
          </div>
        )}

        {/* Duration filters */}
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-1">
            <label className="text-xs text-subtle mb-1 block">Min Duration (s, 0=off)</label>
            <input
              type="number"
              value={shortsConfig.minDuration}
              onChange={(e) => setShortsConfig({ minDuration: Number(e.target.value) })}
              min={0}
              className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-subtle mb-1 block">Max Duration (s, 0=off)</label>
            <input
              type="number"
              value={shortsConfig.maxDuration}
              onChange={(e) => setShortsConfig({ maxDuration: Number(e.target.value) })}
              min={0}
              className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60"
            />
          </div>
        </div>

        <Separator />

        {/* Enhancement toggles */}
        <div className="flex flex-col gap-3">
          <p className="text-xs text-subtle uppercase tracking-wider">Enhancements</p>

          {/* Blur Background */}
          <div className="flex items-center gap-2">
            <Checkbox
              id="blur-bg"
              checked={shortsConfig.addBlurBackground}
              onCheckedChange={(v) => setShortsConfig({ addBlurBackground: !!v })}
            />
            <Label htmlFor="blur-bg" className="text-sm cursor-pointer">
              Blur Background
            </Label>
          </div>

          {/* Word-by-word Captions */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <Checkbox
                id="captions"
                checked={shortsConfig.addCaptions}
                onCheckedChange={(v) => setShortsConfig({ addCaptions: !!v })}
              />
              <Label htmlFor="captions" className="text-sm cursor-pointer">
                Word-by-word Captions
              </Label>
            </div>
            {shortsConfig.addCaptions && (
              <input
                type="text"
                value={shortsConfig.transcriptPath}
                onChange={(e) => setShortsConfig({ transcriptPath: e.target.value })}
                placeholder="/path/to/transcript.json"
                className="w-full rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60 ml-6"
              />
            )}
          </div>

          {/* Language selector */}
          <div className="flex items-center gap-3">
            <label className="text-sm text-foreground min-w-[6rem]">Language</label>
            <select
              value={shortsConfig.language}
              onChange={(e) => setShortsConfig({ language: e.target.value as 'pt' | 'en' | 'es' })}
              className="rounded-lg bg-surface border border-border px-3 py-1.5 text-sm focus:outline-none focus:border-brand/60"
            >
              <option value="pt">PT</option>
              <option value="en">EN</option>
              <option value="es">ES</option>
            </select>
          </div>

          {/* Generate Thumbnails */}
          <div className="flex items-center gap-2">
            <Checkbox
              id="thumbnails"
              checked={shortsConfig.generateThumbnails}
              onCheckedChange={(v) => setShortsConfig({ generateThumbnails: !!v })}
            />
            <Label htmlFor="thumbnails" className="text-sm cursor-pointer">
              Generate Thumbnails
            </Label>
          </div>
        </div>

        {/* Error */}
        {error && <p className="text-destructive text-sm">{error}</p>}

        {/* Submit button */}
        <Button
          onClick={handleCreate}
          disabled={isRunning || !input.trim() || !output.trim()}
          className="w-full"
        >
          Create Shorts
        </Button>

        {/* Job status + log viewer */}
        {activeJob && (
          <div className="flex flex-col gap-2 mt-1">
            <div className="flex items-center gap-2">
              <span className="text-xs text-subtle">Status</span>
              <StatusBadge status={activeJob.status} />
            </div>
            {activeJob.logs.length > 0 && (
              <ScrollArea className="h-40 rounded-lg bg-surface-inset border border-border p-3">
                <div className="font-mono text-xs space-y-0.5">
                  {activeJob.logs.map((line, i) => (
                    <p key={i} className="text-caption leading-relaxed whitespace-pre-wrap">
                      {line}
                    </p>
                  ))}
                </div>
              </ScrollArea>
            )}
          </div>
        )}
      </div>

      {/* Results */}
      {activeJobId && (
        <div className="flex flex-col gap-3">
          <h2 className="text-sm font-medium text-foreground">Results</h2>
          <ShortsResultList jobId={activeJobId} language={shortsConfig.language} />
        </div>
      )}
    </div>
  );
}
