import * as React from 'react'
import { cn } from '@/lib/utils'

interface SettingRowProps {
  label: React.ReactNode
  description?: React.ReactNode
  className?: string
  children: React.ReactNode
}

function SettingRow({ label, description, className, children }: SettingRowProps) {
  return (
    <div
      data-slot="setting-row"
      className={cn('flex items-center justify-between gap-3', className)}
    >
      <div className="flex flex-col min-w-0">
        <p className="text-xs font-medium text-prose">{label}</p>
        {description && (
          <p data-slot="setting-row-description" className="text-xs text-subtle">
            {description}
          </p>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  )
}

export { SettingRow }
export type { SettingRowProps }
