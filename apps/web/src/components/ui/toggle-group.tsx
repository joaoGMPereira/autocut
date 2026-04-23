import * as React from 'react'
import { cn } from '@/lib/utils'

interface ToggleGroupOption<K extends string> {
  key: K
  label: string
}

interface ToggleGroupProps<K extends string> {
  options: ToggleGroupOption<K>[]
  value: Record<K, boolean>
  onChange: (key: K, next: boolean) => void
  className?: string
}

function ToggleGroup<K extends string>({
  options,
  value,
  onChange,
  className,
}: ToggleGroupProps<K>) {
  return (
    <div
      data-slot="toggle-group"
      className={cn('flex gap-2', className)}
    >
      {options.map((opt) => {
        const active = value[opt.key]
        return (
          <button
            key={opt.key}
            type="button"
            onClick={() => onChange(opt.key, !active)}
            aria-pressed={active}
            className={cn(
              'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
              active
                ? 'bg-brand text-brand-foreground font-medium'
                : 'bg-surface text-subtle hover:bg-surface/80',
            )}
          >
            {opt.label}
          </button>
        )
      })}
    </div>
  )
}

export { ToggleGroup }
export type { ToggleGroupProps, ToggleGroupOption }
