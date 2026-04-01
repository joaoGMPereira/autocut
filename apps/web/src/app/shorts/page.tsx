'use client';

import { Anton } from 'next/font/google';
import { useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { useProcessorStore } from '@/store/processorStore';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import type { JobStatus } from '@/store/processorStore';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

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
    <span className="bg-[#1A1A26] text-[#5C5C80] text-xs px-2 py-0.5 rounded-md">
      idle
    </span>
  );
}

export default function ShortsPage() {
  const { goUrl } = useAppStore();
  const { shortsJobs, startShorts } = useProcessorStore();

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

  const handleCreate = async () => {
    setError(null);
    try {
      const id = await startShorts(goUrl, { input, output, width, height });
      setActiveJobId(id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start');
    }
  };

  const isRunning = activeJob?.status === 'running';

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        SHORTS
      </h1>

      <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-4">
        {/* Input path */}
        <div className="flex flex-col gap-1">
          <label className="text-xs text-[#5C5C80] mb-1 block">Input Path</label>
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="/path/to/input.mp4"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
          />
        </div>

        {/* Output path */}
        <div className="flex flex-col gap-1">
          <label className="text-xs text-[#5C5C80] mb-1 block">Output Path</label>
          <input
            type="text"
            value={output}
            onChange={(e) => setOutput(e.target.value)}
            placeholder="/path/to/output/"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
          />
        </div>

        {/* Aspect ratio preset */}
        <div className="flex flex-col gap-1">
          <label className="text-xs text-[#5C5C80] mb-1 block">Aspect Ratio</label>
          <select
            value={preset}
            onChange={(e) => setPreset(Number(e.target.value))}
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
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
              <label className="text-xs text-[#5C5C80] mb-1 block">Width (px)</label>
              <input
                type="number"
                value={customWidth}
                onChange={(e) => setCustomWidth(Number(e.target.value))}
                min={1}
                className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-[#5C5C80] mb-1 block">Height (px)</label>
              <input
                type="number"
                value={customHeight}
                onChange={(e) => setCustomHeight(Number(e.target.value))}
                min={1}
                className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
              />
            </div>
          </div>
        )}

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
              <span className="text-xs text-[#5C5C80]">Status</span>
              <StatusBadge status={activeJob.status} />
            </div>
            {activeJob.logs.length > 0 && (
              <ScrollArea className="h-40 rounded-lg bg-[#0E0E1A] border border-border p-3">
                <div className="font-mono text-xs space-y-0.5">
                  {activeJob.logs.map((line, i) => (
                    <p key={i} className="text-[#A0A0C0] leading-relaxed whitespace-pre-wrap">
                      {line}
                    </p>
                  ))}
                </div>
              </ScrollArea>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
