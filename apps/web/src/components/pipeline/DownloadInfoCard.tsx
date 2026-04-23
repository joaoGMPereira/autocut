'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { SectionPanel } from '@/components/ui/section-panel';

export function DownloadInfoCard() {
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);
  const videoInfo = usePipelineStore((s) => s.videoInfo);

  if (!phaseProgress || phaseProgress.phase !== 'download') return null;

  const pct = Math.round(phaseProgress.percentDone);
  const speed = phaseProgress.speedKbs != null
    ? phaseProgress.speedKbs > 1024
      ? `${(phaseProgress.speedKbs / 1024).toFixed(1)} MB/s`
      : `${phaseProgress.speedKbs.toFixed(0)} KB/s`
    : null;
  const eta = phaseProgress.etaSec != null
    ? phaseProgress.etaSec > 60
      ? `${Math.floor(phaseProgress.etaSec / 60)}m ${phaseProgress.etaSec % 60}s`
      : `${phaseProgress.etaSec}s`
    : null;

  return (
    <SectionPanel size="sm" title="Download">
      {videoInfo && (
        <div className="flex items-center gap-3">
          {videoInfo.thumbnailUrl && (
            <img
              src={videoInfo.thumbnailUrl}
              alt={videoInfo.title}
              className="h-12 w-20 rounded object-cover shrink-0"
            />
          )}
          {videoInfo.title && (
            <p className="text-xs text-foreground line-clamp-2 leading-snug">{videoInfo.title}</p>
          )}
        </div>
      )}

      <div className="w-full h-1.5 bg-brand/20 rounded-full overflow-hidden">
        <div
          className="h-full bg-brand rounded-full transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
      <div className="flex items-center justify-between text-xs text-subtle">
        <span>{pct}%</span>
        <span>{speed ?? ''}</span>
        <span>{eta ? `ETA ${eta}` : ''}</span>
      </div>
    </SectionPanel>
  );
}
