'use client';

import type { ReactNode } from 'react';
import { useState } from 'react';
import { cn } from '@/lib/utils';

interface ToggleSectionProps {
  title: string;
  statusText?: string; // e.g. "Ativado - Modo Sutil" | "Desativado"
  icon: ReactNode;
  enabled: boolean;
  onToggle: (enabled: boolean) => void;
  children?: ReactNode;
}

export function ToggleSection({
  title,
  statusText,
  icon,
  enabled,
  onToggle,
  children,
}: ToggleSectionProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="rounded-xl border border-border bg-[#12121C]">
      {/* Header row */}
      <div className="flex items-center justify-between gap-3 px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="shrink-0 text-base">{icon}</span>
          <div className="flex min-w-0 flex-col">
            <span className="text-sm font-semibold text-[#F0F0F8]">{title}</span>
            {statusText && (
              <span
                className={cn(
                  'text-xs',
                  enabled ? 'text-[#6BCB8B]' : 'text-[#5C5C80]',
                )}
              >
                {statusText}
              </span>
            )}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {/* Toggle switch */}
          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            onClick={() => onToggle(!enabled)}
            className={cn(
              'relative h-6 w-11 rounded-full transition-colors',
              enabled ? 'bg-[#6BCB8B]' : 'bg-[#2A2A3C]',
            )}
          >
            <span
              className={cn(
                'absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform',
                enabled ? 'translate-x-5' : 'translate-x-0.5',
              )}
            />
          </button>

          {/* Expand chevron (only if has children) */}
          {children && (
            <button
              type="button"
              onClick={() => setExpanded((v) => !v)}
              className="text-[#5C5C80] transition-colors hover:text-[#A0A0B8]"
              aria-label={expanded ? 'Recolher' : 'Expandir'}
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="currentColor"
                className={cn('transition-transform', expanded ? 'rotate-180' : '')}
              >
                <path d="M4 6l4 4 4-4" stroke="currentColor" strokeWidth="1.5" fill="none" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Expanded content */}
      {children && expanded && (
        <div className="border-t border-border px-4 py-3">
          {children}
        </div>
      )}
    </div>
  );
}
