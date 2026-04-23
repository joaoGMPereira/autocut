import { Badge } from '@/components/ui/badge';

export interface EngagementScoreBarProps {
  score: number;
  factors: string[];
}

function getScoreColor(score: number): { bar: string; text: string } {
  if (score < 40) return { bar: 'bg-destructive', text: 'text-destructive' };
  if (score <= 70) return { bar: 'bg-warning', text: 'text-warning' };
  return { bar: 'bg-success', text: 'text-success' };
}

export function EngagementScoreBar({ score, factors }: EngagementScoreBarProps) {
  const clamped = Math.min(100, Math.max(0, score));
  const { bar, text } = getScoreColor(clamped);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between">
        <span className="text-xs text-subtle">Engagement</span>
        <span className={`text-xs font-mono font-medium ${text}`}>{clamped.toFixed(1)}</span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${bar}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
      {factors.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-0.5">
          {factors.map((f) => (
            <Badge key={f} variant="outline" className="text-[10px] px-1.5 py-0 font-mono">
              {f}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}
