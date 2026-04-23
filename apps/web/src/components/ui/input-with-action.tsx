import * as React from 'react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface InputWithActionProps {
  value: string
  onValueChange: (value: string) => void
  onSubmit: () => void
  actionLabel: React.ReactNode
  actionDisabled?: boolean
  placeholder?: string
  inputClassName?: string
  actionVariant?: React.ComponentProps<typeof Button>['variant']
  actionSize?: React.ComponentProps<typeof Button>['size']
  className?: string
}

function InputWithAction({
  value,
  onValueChange,
  onSubmit,
  actionLabel,
  actionDisabled,
  placeholder,
  inputClassName,
  actionVariant = 'outline',
  actionSize = 'sm',
  className,
}: InputWithActionProps) {
  return (
    <div
      data-slot="input-with-action"
      className={cn('flex gap-2', className)}
    >
      <Input
        type="text"
        value={value}
        onChange={(e) => onValueChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !actionDisabled) {
            onSubmit()
          }
        }}
        placeholder={placeholder}
        className={cn('flex-1', inputClassName)}
      />
      <Button
        type="button"
        onClick={onSubmit}
        disabled={actionDisabled}
        variant={actionVariant}
        size={actionSize}
        className="text-xs"
      >
        {actionLabel}
      </Button>
    </div>
  )
}

export { InputWithAction }
export type { InputWithActionProps }
