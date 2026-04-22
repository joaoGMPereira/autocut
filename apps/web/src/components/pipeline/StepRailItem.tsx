'use client';

import type { StepStatus } from '@/types/pipeline';

interface StepRailItemProps {
  label: string;
  status: StepStatus;
  isLast?: boolean;
}

const ringClass: Record<StepStatus, string> = {
  idle: 'border-2 border-zinc-700 bg-transparent',
  running: 'border-2 border-brand bg-brand/10 animate-pulse',
  gate: 'border-2 border-brand bg-brand/10',
  done: 'border-2 border-emerald-400 bg-emerald-400/20',
  error: 'border-2 border-red-500 bg-red-500/20',
};

const dotClass: Record<StepStatus, string> = {
  idle: 'bg-zinc-600',
  running: 'bg-brand',
  gate: 'bg-brand',
  done: 'bg-emerald-400',
  error: 'bg-red-500',
};

const labelClass: Record<StepStatus, string> = {
  idle: 'text-zinc-500',
  running: 'text-brand font-medium',
  gate: 'text-brand font-medium',
  done: 'text-emerald-400',
  error: 'text-red-400',
};

export function StepRailItem({ label, status, isLast }: StepRailItemProps) {
  return (
    <div className="flex flex-col items-start">
      <div className="flex items-center gap-3 py-2 px-4 w-full">
        {/* Status ring */}
        <div className={`w-5 h-5 rounded-full flex items-center justify-center shrink-0 ${ringClass[status]}`}>
          <div className={`w-2 h-2 rounded-full ${dotClass[status]}`} />
        </div>
        {/* Label */}
        <span className={`text-sm truncate ${labelClass[status]}`}>{label}</span>
      </div>
      {/* Connector line */}
      {!isLast && (
        <div className="w-px h-4 bg-zinc-700 ml-[28px]" />
      )}
    </div>
  );
}
