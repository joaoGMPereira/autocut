import * as React from "react"

import { cn } from "@/lib/utils"

type InfoBannerProps = React.HTMLAttributes<HTMLDivElement>

function InfoBanner({ className, children, ...props }: InfoBannerProps) {
  return (
    <div
      data-slot="info-banner"
      className={cn(
        "rounded-md border border-border bg-card px-4 py-3 text-xs text-prose",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}

export { InfoBanner }
export type { InfoBannerProps }
