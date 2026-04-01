'use client';

import { useState } from 'react';
import { StepCard, type StepStatus } from '@/components/pipeline/StepCard';

interface ThumbnailStepProps {
  status: StepStatus;
}

export function ThumbnailStep({ status }: ThumbnailStepProps) {
  const [strategy, setStrategy] = useState('auto');

  return (
    <StepCard title="Thumbnail" stepNumber={6} status={status}>
      <div className="flex flex-col gap-3">
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-[#5C5C80]">Strategy</span>
          <select
            value={strategy}
            onChange={(e) => setStrategy(e.target.value)}
            className="rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
          >
            <option value="auto">Auto</option>
            <option value="frame">Best frame</option>
            <option value="custom">Custom</option>
          </select>
        </label>
        <p className="text-xs text-[#5C5C80]">Thumbnail generation (placeholder).</p>
      </div>
    </StepCard>
  );
}
