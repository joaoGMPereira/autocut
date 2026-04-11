import type { VideoInfoData } from '@/types/download';

interface Props {
  info: VideoInfoData;
}

function formatDuration(secs: number): string {
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
}

export function VideoInfoCard({ info }: Props) {
  return (
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      <div className="aspect-video w-full bg-zinc-800">
        <img
          src={info.thumbnailUrl}
          alt={info.title}
          className="w-full h-full object-cover"
        />
      </div>
      <div className="p-3 space-y-1">
        <p className="text-sm font-medium text-foreground leading-snug line-clamp-2">
          {info.title}
        </p>
        <p className="text-xs text-zinc-400">{formatDuration(info.durationSec)}</p>
      </div>
    </div>
  );
}
