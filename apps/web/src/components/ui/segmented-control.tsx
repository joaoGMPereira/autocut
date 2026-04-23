import * as React from 'react'
import { cn } from '@/lib/utils'

interface SegmentedOption<T extends string> {
  value: T
  label: string
  description?: string
}

interface SegmentedControlProps<T extends string> {
  options: SegmentedOption<T>[]
  value: T
  onChange: (value: T) => void
  variant?: 'flat' | 'card'
  className?: string
}

// Tailwind cannot purge interpolated class names — use a static lookup.
const GRID_COLS: Record<number, string> = {
  2: 'grid-cols-2',
  3: 'grid-cols-3',
  4: 'grid-cols-4',
}

function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  variant = 'flat',
  className,
}: SegmentedControlProps<T>) {
  if (variant === 'card') {
    const colClass = GRID_COLS[options.length] ?? 'grid-cols-2'
    return (
      <div
        data-slot="segmented-control"
        data-variant="card"
        className={cn('grid gap-2', colClass, className)}
      >
        {options.map((opt) => (
          <button
            key={opt.value}
            type="button"
            onClick={() => onChange(opt.value)}
            className={cn(
              'flex flex-col items-start rounded-lg border p-3 text-left transition-all',
              opt.value === value
                ? 'border-brand bg-surface ring-1 ring-brand'
                : 'border-border bg-surface hover:border-border/60',
            )}
            aria-pressed={opt.value === value}
          >
            <span className="text-xs font-medium text-prose">{opt.label}</span>
            {opt.description && (
              <span className="text-xs text-subtle">{opt.description}</span>
            )}
          </button>
        ))}
      </div>
    )
  }

  return (
    <div
      data-slot="segmented-control"
      data-variant="flat"
      className={cn('flex gap-2', className)}
    >
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          onClick={() => onChange(opt.value)}
          className={cn(
            'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
            opt.value === value
              ? 'bg-brand text-brand-foreground font-medium'
              : 'bg-surface text-subtle hover:bg-surface/80',
          )}
          aria-pressed={opt.value === value}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

export { SegmentedControl }
export type { SegmentedControlProps, SegmentedOption }
