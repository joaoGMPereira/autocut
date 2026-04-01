'use client';

import { cn } from '@/lib/utils';

interface DurationChipsProps {
  label: string;
  presets: number[]; // minutes
  value: number; // minutes
  onChange: (minutes: number) => void;
}

export function DurationChips({ label, presets, value, onChange }: DurationChipsProps) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-[#F0F0F8]">{label}</span>
        <span className="text-xs text-[#5C5C80]">Atual: {value}min</span>
      </div>
      <div className="flex flex-wrap gap-2">
        {presets.map((preset) => (
          <button
            key={preset}
            type="button"
            onClick={() => onChange(preset)}
            className={cn(
              'rounded-md border px-3 py-1.5 text-sm transition-all',
              value === preset
                ? 'border-[#F0F0F8] bg-[#F0F0F8]/10 text-[#F0F0F8]'
                : 'border-border bg-[#12121C] text-[#A0A0B8] hover:border-[#A0A0B8]/50',
            )}
          >
            {preset}min
          </button>
        ))}
      </div>
    </div>
  );
}
