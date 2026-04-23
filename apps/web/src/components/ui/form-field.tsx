"use client"

import * as React from "react"

import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

interface FormFieldProps {
  label: React.ReactNode
  description?: React.ReactNode
  error?: React.ReactNode
  counter?: React.ReactNode
  required?: boolean
  htmlFor?: string
  className?: string
  children: React.ReactElement<{
    id?: string
    "aria-describedby"?: string
    "aria-invalid"?: boolean
  }>
}

function FormField({
  label,
  description,
  error,
  counter,
  required,
  htmlFor,
  className,
  children,
}: FormFieldProps) {
  const generatedId = React.useId()
  const id = htmlFor ?? children.props.id ?? generatedId

  const showDescription = !error && description != null
  const descriptionId = showDescription ? `${id}-description` : undefined
  const errorId = error != null ? `${id}-error` : undefined
  const ariaDescribedBy =
    [descriptionId, errorId].filter(Boolean).join(" ") || undefined

  const child = React.cloneElement(children, {
    id,
    "aria-describedby":
      ariaDescribedBy ?? children.props["aria-describedby"],
    "aria-invalid":
      error != null ? true : children.props["aria-invalid"],
  })

  const labelNode = (
    <Label htmlFor={id} className="text-xs font-medium text-subtle">
      {label}
      {required && <span className="text-destructive ml-0.5">*</span>}
    </Label>
  )

  return (
    <div
      data-slot="form-field"
      className={cn("flex flex-col gap-1.5", className)}
    >
      {counter != null ? (
        <div className="flex items-center justify-between">
          {labelNode}
          <span className="text-xs text-caption">{counter}</span>
        </div>
      ) : (
        labelNode
      )}
      {child}
      {error != null ? (
        <p id={errorId} className="text-xs text-destructive">
          {error}
        </p>
      ) : description != null ? (
        <p id={descriptionId} className="text-xs text-subtle">
          {description}
        </p>
      ) : null}
    </div>
  )
}

export { FormField }
export type { FormFieldProps }
