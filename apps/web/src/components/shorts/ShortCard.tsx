import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import type { ShortResult } from '@/store/processorStore';
import { EngagementScoreBar } from './EngagementScoreBar';

interface ShortCardProps {
  result: ShortResult;
  language: string;
}

function basename(filePath: string): string {
  return filePath.split('/').pop() ?? filePath;
}

export function ShortCard({ result, language }: ShortCardProps) {
  return (
    <div className="rounded-xl bg-card border border-border p-4 flex flex-col gap-3">
      {result.thumbnail_path && (
        <img
          src={result.thumbnail_path}
          alt="thumbnail"
          className="w-full rounded-lg object-cover aspect-video"
        />
      )}

      <div className="flex items-center gap-2 flex-wrap">
        <span className="font-mono text-xs text-subtle truncate max-w-[200px]">
          {basename(result.output)}
        </span>
        {result.duration_sec !== undefined && (
          <Badge variant="outline" className="text-[10px] font-mono px-1.5 py-0">
            {result.duration_sec.toFixed(1)}s
          </Badge>
        )}
        <Badge variant="info" className="text-[10px] px-1.5 py-0">
          {language.toUpperCase()}
        </Badge>
      </div>

      <Separator />

      <EngagementScoreBar
        score={result.engagement_score}
        factors={result.engagement_factors}
      />
    </div>
  );
}
