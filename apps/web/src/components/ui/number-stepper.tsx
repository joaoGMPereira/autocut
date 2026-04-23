'use client'

import * as React from 'react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface NumberStepperBaseProps {
  min?: number
  max?: number
  step?: number
  size?: 'sm' | 'default'
  inputClassName?: string
  className?: string
  ariaLabel?: string
  disabled?: boolean
}

interface NumberStepperRequiredProps extends NumberStepperBaseProps {
  value: number
  onValueChange: (n: number) => void
  allowEmpty?: false
}

interface NumberStepperNullableProps extends NumberStepperBaseProps {
  value: number | null
  onValueChange: (n: number | null) => void
  allowEmpty: true
  placeholder?: string
}

type NumberStepperProps = NumberStepperRequiredProps | NumberStepperNullableProps

function NumberStepper({
  value,
  onValueChange,
  min,
  max,
  step = 1,
  size = 'sm',
  inputClassName,
  className,
  ariaLabel,
  disabled,
  ...rest
}: NumberStepperProps) {
  const allowEmpty = (rest as NumberStepperNullableProps).allowEmpty === true
  const placeholder = allowEmpty ? (rest as NumberStepperNullableProps).placeholder : undefined

  const buttonSize = size === 'sm' ? 'icon-sm' : 'icon'

  const clamp = (n: number): number => {
    let result = n
    if (min !== undefined) result = Math.max(min, result)
    if (max !== undefined) result = Math.min(max, result)
    return result
  }

  const handleDecrement = () => {
    if (value === null) return
    const next = clamp(value - step)
    ;(onValueChange as (n: number) => void)(next)
  }

  const handleIncrement = () => {
    if (value === null) {
      const start = min !== undefined ? min : 0
      ;(onValueChange as (n: number) => void)(start)
      return
    }
    const next = clamp(value + step)
    ;(onValueChange as (n: number) => void)(next)
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const parsed = parseInt(e.target.value, 10)
    if (isNaN(parsed)) {
      if (allowEmpty) {
        ;(onValueChange as (n: number | null) => void)(null)
      }
      return
    }
    ;(onValueChange as (n: number) => void)(clamp(parsed))
  }

  const isDecrementDisabled =
    disabled || value === null || (min !== undefined && value <= min)
  const isIncrementDisabled =
    disabled || (value !== null && max !== undefined && value >= max)

  return (
    <div
      data-slot="number-stepper"
      className={cn('flex items-center gap-1', className)}
    >
      <Button
        type="button"
        variant="outline"
        size={buttonSize}
        disabled={isDecrementDisabled}
        onClick={handleDecrement}
        aria-label={ariaLabel ? `Decrease ${ariaLabel}` : 'Decrease'}
      >
        −
      </Button>
      <Input
        type="number"
        value={value === null ? '' : value}
        onChange={handleChange}
        placeholder={placeholder}
        disabled={disabled}
        min={min}
        max={max}
        className={cn('w-20 text-center [appearance:textfield]', inputClassName)}
      />
      <Button
        type="button"
        variant="outline"
        size={buttonSize}
        disabled={isIncrementDisabled}
        onClick={handleIncrement}
        aria-label={ariaLabel ? `Increase ${ariaLabel}` : 'Increase'}
      >
        +
      </Button>
    </div>
  )
}

export { NumberStepper }
export type { NumberStepperProps }
