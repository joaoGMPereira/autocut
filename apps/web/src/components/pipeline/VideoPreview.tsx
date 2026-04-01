'use client';

import { useRef, useState, useCallback, useEffect } from 'react';
import { cn } from '@/lib/utils';

interface VideoPreviewProps {
  src: string;
  className?: string;
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${String(s).padStart(2, '0')}`;
}

export function VideoPreview({ src, className }: VideoPreviewProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const timelineRef = useRef<HTMLDivElement>(null);
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  function togglePlayPause() {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      void video.play();
    } else {
      video.pause();
    }
  }

  const handleTimelineClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const video = videoRef.current;
    const bar = timelineRef.current;
    if (!video || !bar || duration === 0) return;
    const rect = bar.getBoundingClientRect();
    const ratio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    video.currentTime = ratio * duration;
  }, [duration]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onEnded = () => setPlaying(false);
    const onTimeUpdate = () => setCurrentTime(video.currentTime);
    const onDurationChange = () => setDuration(video.duration || 0);
    const onLoadedMetadata = () => setDuration(video.duration || 0);

    video.addEventListener('play', onPlay);
    video.addEventListener('pause', onPause);
    video.addEventListener('ended', onEnded);
    video.addEventListener('timeupdate', onTimeUpdate);
    video.addEventListener('durationchange', onDurationChange);
    video.addEventListener('loadedmetadata', onLoadedMetadata);

    return () => {
      video.removeEventListener('play', onPlay);
      video.removeEventListener('pause', onPause);
      video.removeEventListener('ended', onEnded);
      video.removeEventListener('timeupdate', onTimeUpdate);
      video.removeEventListener('durationchange', onDurationChange);
      video.removeEventListener('loadedmetadata', onLoadedMetadata);
    };
  }, [src]);

  const progress = duration > 0 ? (currentTime / duration) * 100 : 0;

  return (
    <div className={cn('flex flex-col overflow-hidden rounded-xl border border-border bg-black', className)}>
      {/* Video + play/pause overlay */}
      <div
        className="relative flex aspect-video cursor-pointer items-center justify-center"
        onClick={togglePlayPause}
      >
        <video
          ref={videoRef}
          src={src}
          controls={false}
          className="h-full w-full object-contain"
          preload="metadata"
        />
        {/* Play/pause button centered */}
        <div
          className={cn(
            'pointer-events-none absolute flex h-12 w-12 items-center justify-center rounded-full bg-black/60 text-white transition-opacity duration-150',
            playing ? 'opacity-0' : 'opacity-100',
          )}
          aria-hidden="true"
        >
          <span className="text-xl leading-none">{playing ? '⏸' : '▶'}</span>
        </div>
      </div>

      {/* Controls bar */}
      <div className="flex flex-col gap-1.5 bg-[#0D0D15] px-3 py-2">
        {/* Timeline */}
        <div
          ref={timelineRef}
          className="relative h-1.5 w-full cursor-pointer rounded-full bg-[#1A1A26]"
          onClick={handleTimelineClick}
          role="slider"
          aria-label="Progresso do vídeo"
          aria-valuenow={currentTime}
          aria-valuemin={0}
          aria-valuemax={duration}
        >
          <div
            className="h-full rounded-full bg-[#00D4FF] transition-[width] duration-100"
            style={{ width: `${progress}%` }}
          />
        </div>

        {/* Time display + play/pause button */}
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={togglePlayPause}
            className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-white hover:text-[#00D4FF] transition-colors"
            aria-label={playing ? 'Pausar' : 'Reproduzir'}
          >
            <span className="text-sm leading-none">{playing ? '⏸' : '▶'}</span>
          </button>

          <span className="font-mono text-xs text-[#A0A0B8]">
            {formatTime(currentTime)}
            {' / '}
            {duration > 0 ? formatTime(duration) : '--:--'}
          </span>
        </div>
      </div>
    </div>
  );
}
