'use client';

import { Anton } from 'next/font/google';
import { useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { useProcessorStore } from '@/store/processorStore';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import type { JobStatus } from '@/store/processorStore';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

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

export default function OptimizerPage() {
  const { goUrl } = useAppStore();
  const { optimizeJobs, startOptimize } = useProcessorStore();

  const [input, setInput] = useState('');
  const [output, setOutput] = useState('');
  const [threshold, setThreshold] = useState(-40);
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const activeJob = activeJobId ? optimizeJobs[activeJobId] : null;
  const isRunning = activeJob?.status === 'running';

  const handleOptimize = async () => {
    setError(null);
    try {
      const id = await startOptimize(goUrl, {
        input,
        output,
        silence_threshold: threshold,
      });
      setActiveJobId(id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start');
    }
  };

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        OPTIMIZER
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
            placeholder="/path/to/output.mp4"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
          />
        </div>

        {/* Silence threshold slider */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <label className="text-xs text-[#5C5C80]">Silence Threshold</label>
            <span className="font-mono text-sm text-[#00D4FF]">{threshold} dB</span>
          </div>
          <input
            type="range"
            min="-60"
            max="-20"
            step="1"
            value={threshold}
            onChange={(e) => setThreshold(Number(e.target.value))}
            className="w-full accent-[#00D4FF]"
          />
          <p className="text-xs text-[#5C5C80]">Silence below this dB will be removed</p>
        </div>

        {/* Error */}
        {error && <p className="text-destructive text-sm">{error}</p>}

        {/* Submit button */}
        <Button
          onClick={handleOptimize}
          disabled={isRunning || !input.trim() || !output.trim()}
          className="w-full"
        >
          Optimize
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
