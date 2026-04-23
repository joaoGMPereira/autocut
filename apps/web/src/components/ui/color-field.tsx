'use client'

import * as React from 'react'
import { cn } from '@/lib/utils'
import { Label } from '@/components/ui/label'

const COLOR_TOKENS = {
  brand:       { label: 'Brand',       hex: '#C6F24E', bgClass: 'bg-brand'       },
  success:     { label: 'Success',     hex: '#34D399', bgClass: 'bg-success'     },
  warning:     { label: 'Warning',     hex: '#FBBF24', bgClass: 'bg-warning'     },
  info:        { label: 'Info',        hex: '#60A5FA', bgClass: 'bg-info'        },
  violet:      { label: 'Violet',      hex: '#A78BFA', bgClass: 'bg-violet'      },
  foreground:  { label: 'White',       hex: '#F5F5FA', bgClass: 'bg-foreground'  },
  background:  { label: 'Black',       hex: '#0E0E16', bgClass: 'bg-background'  },
  destructive: { label: 'Red',         hex: '#8B2020', bgClass: 'bg-destructive' },
} as const

type ColorToken = keyof typeof COLOR_TOKENS

const DEFAULT_TOKENS = Object.keys(COLOR_TOKENS) as ColorToken[]

interface ColorFieldProps {
  value: string
  onValueChange: (hex: string, token: ColorToken) => void
  tokens?: ColorToken[]
  label?: string
  id?: string
  className?: string
  size?: 'sm' | 'default'
}

function ColorField({
  value,
  onValueChange,
  tokens = DEFAULT_TOKENS,
  label,
  id,
  className,
  size = 'default',
}: ColorFieldProps) {
  const sizeClass = size === 'sm' ? 'h-6 w-6' : 'h-8 w-8'

  return (
    <div
      data-slot="color-field"
      id={id}
      className={cn('flex flex-col gap-2', className)}
    >
      {label && (
        <Label className="text-xs font-medium text-subtle">{label}</Label>
      )}
      <div className="flex flex-wrap gap-2">
        {tokens.map((tokenKey) => {
          const token = COLOR_TOKENS[tokenKey]
          const isSelected =
            value.toUpperCase() === token.hex.toUpperCase()
          return (
            <button
              key={tokenKey}
              type="button"
              aria-label={token.label}
              title={token.label}
              onClick={() => onValueChange(token.hex, tokenKey)}
              className={cn(
                token.bgClass,
                sizeClass,
                'rounded-md border border-border cursor-pointer shrink-0',
                isSelected && 'ring-2 ring-offset-1 ring-brand ring-offset-background',
              )}
            />
          )
        })}
      </div>
    </div>
  )
}

export { ColorField, COLOR_TOKENS }
export type { ColorFieldProps, ColorToken }
