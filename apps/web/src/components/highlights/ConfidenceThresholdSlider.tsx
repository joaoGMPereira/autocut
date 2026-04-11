'use client';

import { Slider } from '@/components/ui/slider';

interface ConfidenceThresholdSliderProps {
  value: number;
  onChange: (v: number) => void;
}

export function ConfidenceThresholdSlider({ value, onChange }: ConfidenceThresholdSliderProps) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-zinc-300">
          Confidence Threshold:{' '}
          <span className="font-mono text-blue-400">{value.toFixed(2)}</span>
        </span>
      </div>
      <Slider
        min={0.3}
        max={0.9}
        step={0.05}
        value={[value]}
        onValueChange={(vals) => onChange(vals[0] ?? value)}
        className="w-full"
      />
      <p className="text-[11px] text-zinc-500">
        Highlights abaixo deste valor serão descartados
      </p>
    </div>
  );
}
