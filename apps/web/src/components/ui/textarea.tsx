import * as React from "react"

import { cn } from "@/lib/utils"

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "w-full min-w-0 rounded-lg bg-surface border border-border px-3 py-2 text-sm text-foreground placeholder:text-subtle",
        "shadow-xs transition-[color,box-shadow,border-color] outline-none",
        "selection:bg-primary selection:text-primary-foreground",
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
        "focus-visible:border-brand/60 focus-visible:ring-brand/30 focus-visible:ring-[3px]",
        "aria-invalid:border-destructive aria-invalid:ring-destructive/30",
        "resize-y field-sizing-fixed",
        className,
      )}
      {...props}
    />
  )
}

export { Textarea }
