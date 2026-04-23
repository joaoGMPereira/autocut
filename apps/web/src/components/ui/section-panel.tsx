import * as React from "react"

import { cn } from "@/lib/utils"

interface SectionPanelProps {
  title: React.ReactNode
  description?: React.ReactNode
  actions?: React.ReactNode
  size?: "sm" | "md"
  className?: string
  children: React.ReactNode
}

function SectionPanel({
  title,
  description,
  actions,
  size = "md",
  className,
  children,
}: SectionPanelProps) {
  const pad = size === "sm" ? "p-3 space-y-3" : "p-4 space-y-4"
  const eyebrow =
    size === "sm"
      ? "text-[10px] font-semibold uppercase tracking-wider text-subtle"
      : "text-xs font-medium uppercase tracking-wider text-prose"

  return (
    <div
      data-slot="section-panel"
      className={cn("rounded-lg border border-border bg-card", pad, className)}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col">
          <p className={eyebrow}>{title}</p>
          {description && (
            <p
              data-slot="section-panel-description"
              className="mt-1 text-xs text-subtle"
            >
              {description}
            </p>
          )}
        </div>
        {actions && (
          <div data-slot="section-panel-actions" className="shrink-0">
            {actions}
          </div>
        )}
      </div>
      {children}
    </div>
  )
}

export { SectionPanel }
export type { SectionPanelProps }
