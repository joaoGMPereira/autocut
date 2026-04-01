'use client';

import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface ModeCardProps {
  title: string;
  description: string;
  icon: ReactNode;
  selected: boolean;
  onClick: () => void;
  features?: string[];
}

export function ModeCard({ title, description, icon, selected, onClick, features }: ModeCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex cursor-pointer items-start gap-3 rounded-xl border p-4 text-left transition-all',
        'bg-[#12121C] hover:border-[#00D4FF]/40',
        selected
          ? 'border-[#00D4FF] bg-[#00D4FF]/5 ring-1 ring-[#00D4FF]/20'
          : 'border-border',
      )}
    >
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-[#1A1A26] text-lg">
        {icon}
      </div>
      <div className="flex flex-col gap-1.5">
        <span className="font-bold text-[#F0F0F8]">{title}</span>
        <span className="text-sm text-[#A0A0B8]">{description}</span>
        {features && features.length > 0 && (
          <ul className="mt-1 flex flex-col gap-0.5">
            {features.map((f) => (
              <li key={f} className="flex items-center gap-1.5 text-xs text-[#6BCB8B]">
                <span aria-hidden>✓</span>
                <span>{f}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </button>
  );
}
