import * as React from "react"

import { cn } from "@/lib/utils"

interface PageHeaderProps {
  title: React.ReactNode
  description?: React.ReactNode
  actions?: React.ReactNode
  className?: string
}

function PageHeader({ title, description, actions, className }: PageHeaderProps) {
  return (
    <div
      data-slot="page-header"
      className={cn("flex items-start justify-between gap-4", className)}
    >
      <div className="flex flex-col">
        <h1 className="font-display text-[32px] font-bold tracking-[-0.02em] text-heading leading-tight">
          {title}
        </h1>
        {description && (
          <p
            data-slot="page-header-description"
            className="text-sm text-subtle mt-1"
          >
            {description}
          </p>
        )}
      </div>
      {actions && (
        <div data-slot="page-header-actions" className="flex items-center gap-2 shrink-0">
          {actions}
        </div>
      )}
    </div>
  )
}

export { PageHeader }
export type { PageHeaderProps }
