import * as React from 'react'
import { cn } from '@/lib/utils'

interface PanelHeaderProps {
  title: React.ReactNode
  actions?: React.ReactNode
  className?: string
}

function PanelHeader({ title, actions, className }: PanelHeaderProps) {
  return (
    <div
      data-slot="panel-header"
      className={cn('flex items-center justify-between', className)}
    >
      <h3 className="text-sm font-medium text-prose">{title}</h3>
      {actions && <div className="shrink-0">{actions}</div>}
    </div>
  )
}

export { PanelHeader }
export type { PanelHeaderProps }
