'use client';

import { useRef } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface ColorInputFieldProps {
  label: string;
  value: string;
  onValueChange: (v: string) => void;
}

export function ColorInputField({ label, value, onValueChange }: ColorInputFieldProps) {
  const colorRef = useRef<HTMLInputElement>(null);

  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={`color-hex-${label}`} className="text-xs text-zinc-400">{label}</Label>
      <div className="flex items-center gap-2">
        {/* Swatch — clicking opens native color picker */}
        <button
          type="button"
          aria-label={`Escolher ${label}`}
          onClick={() => colorRef.current?.click()}
          className="h-8 w-8 rounded border border-zinc-700 shrink-0 cursor-pointer"
          style={{ backgroundColor: value }}
        />
        {/* Hidden native color input — syncs with hex text */}
        <input
          ref={colorRef}
          type="color"
          value={value.startsWith('#') && value.length === 7 ? value : '#000000'}
          onChange={(e) => onValueChange(e.target.value.toUpperCase())}
          className="sr-only"
          tabIndex={-1}
          aria-hidden={true}
        />
        {/* Hex text input */}
        <Input
          id={`color-hex-${label}`}
          data-testid="hex-input"
          value={value}
          onChange={(e) => onValueChange(e.target.value.toUpperCase())}
          className="font-mono text-sm h-8"
          maxLength={9}
        />
      </div>
    </div>
  );
}
