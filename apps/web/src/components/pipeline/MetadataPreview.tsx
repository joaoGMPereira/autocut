'use client';

import { Card } from '@/components/ui/card';

export interface VideoMetadata {
  video_id: string;
  title: string;
  description?: string;
  thumbnail_url: string;
  duration: number; // seconds
  platform?: string;
}

interface MetadataPreviewProps {
  metadata: VideoMetadata;
  compact?: boolean;
}

function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  return `${m}:${String(s).padStart(2, '0')}`;
}

export function MetadataPreview({ metadata, compact = false }: MetadataPreviewProps) {
  return (
    <Card className="flex-row items-center gap-3 border-border bg-[#12121C] p-3">
      <img
        src={metadata.thumbnail_url}
        alt={metadata.title}
        className={`${compact ? 'w-24' : 'w-40'} aspect-video rounded-lg object-cover`}
      />
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <p
          className={`${compact ? 'text-sm' : 'text-base'} font-medium text-[#F0F0F8] ${compact ? 'truncate' : 'line-clamp-2'}`}
        >
          {metadata.title}
        </p>
        <div className="flex items-center gap-2">
          <span className="text-xs text-[#A0A0B8]">{formatDuration(metadata.duration)}</span>
          {metadata.platform && (
            <span className="rounded bg-[#1A1A26] px-1.5 py-0.5 text-xs text-[#A0A0B8]">
              {metadata.platform}
            </span>
          )}
        </div>
      </div>
    </Card>
  );
}
