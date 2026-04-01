'use client';

import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';

export type StepStatus = 'idle' | 'running' | 'done' | 'error' | 'skipped';

function StatusBadge({ status }: { status: StepStatus }) {
  const styles: Record<StepStatus, string> = {
    idle: 'bg-[#1A1A26] text-[#5C5C80] text-xs px-2 py-0.5 rounded-md',
    running: 'bg-blue-500/10 text-blue-400 text-xs px-2 py-0.5 rounded-md',
    done: 'bg-emerald-500/10 text-emerald-400 text-xs px-2 py-0.5 rounded-md',
    error: 'bg-destructive/10 text-destructive text-xs px-2 py-0.5 rounded-md',
    skipped: 'bg-yellow-500/10 text-yellow-400 text-xs px-2 py-0.5 rounded-md',
  };
  return <span className={styles[status]}>{status}</span>;
}

interface StepCardProps {
  title: string;
  stepNumber: number;
  status: StepStatus;
  children: ReactNode;
  onRun?: () => void;
  runDisabled?: boolean;
  runLabel?: string;
}

export function StepCard({ title, stepNumber, status, children, onRun, runDisabled, runLabel }: StepCardProps) {
  return (
    <div className="bg-card border border-border rounded-xl p-5 flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="flex items-center justify-center w-7 h-7 rounded-full bg-[#1A1A26] text-[#5C5C80] text-xs font-medium">
            {stepNumber}
          </span>
          <h2 className="text-base font-semibold text-[#F0F0F8]">{title}</h2>
        </div>
        <StatusBadge status={status} />
      </div>

      {children}

      {onRun && (
        <Button
          onClick={onRun}
          disabled={runDisabled || status === 'running'}
          className="self-start"
          size="sm"
        >
          {runLabel ?? 'Run'}
        </Button>
      )}
    </div>
  );
}
