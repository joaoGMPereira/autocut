import * as React from 'react'
import { cn } from '@/lib/utils'

interface SliderRowProps {
  label: string
  min: number
  max: number
  value: number
  onChange: (value: number) => void
  step?: number
  format?: (value: number) => string
  className?: string
}

function SliderRow({
  label,
  min,
  max,
  value,
  onChange,
  step,
  format,
  className,
}: SliderRowProps) {
  const display = format ? format(value) : String(value)
  return (
    <div
      data-slot="slider-row"
      className={cn('flex items-center gap-3', className)}
    >
      <span className="text-xs font-medium text-subtle shrink-0">{label}</span>
      <input
        type="range"
        aria-label={label}
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="flex-1 accent-brand"
      />
      <span className="text-xs text-subtle shrink-0 tabular-nums min-w-[3ch]">
        {display}
      </span>
    </div>
  )
}

export { SliderRow }
export type { SliderRowProps }
