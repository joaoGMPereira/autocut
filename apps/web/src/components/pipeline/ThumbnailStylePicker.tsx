'use client';

const STYLES = ['default', 'bold', 'minimal', 'dark', 'gradient'] as const;
type ThumbnailStyle = (typeof STYLES)[number];

interface ThumbnailStylePickerProps {
  value: string;
  onChange: (style: string) => void;
}

export function ThumbnailStylePicker({ value, onChange }: ThumbnailStylePickerProps) {
  return (
    <div className="grid grid-cols-3 gap-2">
      {STYLES.map((style) => (
        <button
          key={style}
          type="button"
          onClick={() => onChange(style)}
          className={`rounded-lg border px-3 py-4 text-xs font-medium capitalize transition-colors ${
            value === style
              ? 'border-brand bg-brand/10 text-brand'
              : 'border-border bg-card text-subtle hover:border-caption'
          }`}
        >
          {style}
        </button>
      ))}
    </div>
  );
}
