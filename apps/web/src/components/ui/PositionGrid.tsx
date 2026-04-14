'use client';

const POSITIONS = [
  ['top_left', 'top_center', 'top_right'],
  ['mid_left', 'mid_center', null],
  ['bottom_left', 'bottom_center', 'bottom_right'],
] as const;

const POSITION_LABELS: Record<string, string> = {
  top_left: '↖',
  top_center: '↑',
  top_right: '↗',
  mid_left: '←',
  mid_center: '·',
  bottom_left: '↙',
  bottom_center: '↓',
  bottom_right: '↘',
};

interface PositionGridProps {
  value: string;
  onChange: (pos: string) => void;
}

export function PositionGrid({ value, onChange }: PositionGridProps) {
  return (
    <div className="grid grid-cols-3 gap-1 w-24">
      {POSITIONS.flat().map((pos) => {
        if (!pos) return <div key="empty" />;
        const active = value === pos;
        return (
          <button
            key={pos}
            type="button"
            title={pos.replace('_', ' ')}
            onClick={() => onChange(pos)}
            className={[
              'h-8 w-8 rounded text-sm font-mono flex items-center justify-center transition-colors',
              active
                ? 'bg-[#00D4FF]/20 ring-2 ring-[#00D4FF] text-[#00D4FF]'
                : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700',
            ].join(' ')}
          >
            {POSITION_LABELS[pos]}
          </button>
        );
      })}
    </div>
  );
}
