'use client';

import { useState } from 'react';
import { StepCard, type StepStatus } from '@/components/pipeline/StepCard';

interface UploadStepProps {
  status: StepStatus;
}

export function UploadStep({ status }: UploadStepProps) {
  const [channel, setChannel] = useState('');
  const [privacy, setPrivacy] = useState('private');

  return (
    <StepCard title="Upload" stepNumber={7} status={status}>
      <div className="flex flex-col gap-3">
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-[#5C5C80]">Channel</span>
          <input
            type="text"
            value={channel}
            onChange={(e) => setChannel(e.target.value)}
            placeholder="Select channel..."
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 placeholder:text-[#5C5C80]"
          />
        </label>

        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-[#5C5C80]">Privacy</span>
          <select
            value={privacy}
            onChange={(e) => setPrivacy(e.target.value)}
            className="rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
          >
            <option value="private">Private</option>
            <option value="unlisted">Unlisted</option>
            <option value="public">Public</option>
          </select>
        </label>
        <p className="text-xs text-[#5C5C80]">YouTube upload (placeholder).</p>
      </div>
    </StepCard>
  );
}
