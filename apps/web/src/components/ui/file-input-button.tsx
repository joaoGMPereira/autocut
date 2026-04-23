'use client'

import * as React from 'react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface FileInputButtonProps {
  onFilesSelected: (files: FileList) => void
  accept?: string
  multiple?: boolean
  disabled?: boolean
  children?: React.ReactNode
  variant?: React.ComponentProps<typeof Button>['variant']
  size?: React.ComponentProps<typeof Button>['size']
  className?: string
  inputRef?: React.RefObject<HTMLInputElement>
}

function FileInputButton({
  onFilesSelected,
  accept,
  multiple,
  disabled,
  children = 'Choose file',
  variant = 'outline',
  size = 'sm',
  className,
  inputRef: externalRef,
}: FileInputButtonProps) {
  const internalRef = React.useRef<HTMLInputElement>(null)
  const ref = externalRef ?? internalRef

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0) return
    onFilesSelected(files)
    e.currentTarget.value = ''
  }

  return (
    <div data-slot="file-input-button" className={cn(className)}>
      <Button
        type="button"
        variant={variant}
        size={size}
        disabled={disabled}
        onClick={() => ref.current?.click()}
      >
        {children}
      </Button>
      <input
        ref={ref}
        type="file"
        accept={accept}
        multiple={multiple}
        disabled={disabled}
        className="hidden"
        onChange={handleChange}
        aria-hidden
        tabIndex={-1}
      />
    </div>
  )
}

export { FileInputButton }
export type { FileInputButtonProps }
