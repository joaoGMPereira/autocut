'use client';

import { useState, useRef, useCallback } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { Clip } from '@/types/pipeline';

interface ClipReviewCardProps {
  clip: Clip;
  index: number;
  isSelected: boolean;
  title: string;
  thumbnailText: string;
  onSelectToggle: (clipId: number, selected: boolean) => void;
  onTitleChange: (clipId: number, value: string) => void;
  onThumbnailTextChange: (clipId: number, value: string) => void;
}

function formatDuration(sec: number) {
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export function ClipReviewCard({
  clip,
  index,
  isSelected,
  title,
  thumbnailText,
  onSelectToggle,
  onTitleChange,
  onThumbnailTextChange,
}: ClipReviewCardProps) {
  const [videoOpen, setVideoOpen] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);

  const thumbnailSrc = clip.thumbnail_path
    ? `/files/${encodeURIComponent(clip.thumbnail_path)}`
    : null;
  const videoSrc = clip.file_path
    ? `/files/${encodeURIComponent(clip.file_path)}`
    : null;

  const handleVideoToggle = useCallback(() => {
    setVideoOpen((prev) => {
      if (prev && videoRef.current) videoRef.current.pause();
      return !prev;
    });
  }, []);

  const handleSelectToggle = useCallback(
    () => onSelectToggle(clip.id, !isSelected),
    [clip.id, isSelected, onSelectToggle],
  );

  const handleTitleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => onTitleChange(clip.id, e.target.value),
    [clip.id, onTitleChange],
  );

  const handleThumbnailTextChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => onThumbnailTextChange(clip.id, e.target.value),
    [clip.id, onThumbnailTextChange],
  );

  return (
    <div
      className={`rounded-lg border transition-colors ${
        isSelected ? 'border-brand/40 bg-brand/5' : 'border-border bg-card/60 opacity-60'
      }`}
    >
      {/* Header: index, duration, selection */}
      <div className="flex items-center justify-between px-3 pt-3">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-prose">Clip {index + 1}</span>
          <span className="text-xs text-caption">{formatDuration(clip.duration_sec)}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-subtle">{isSelected ? 'Selecionado' : 'Ignorar'}</span>
          <button
            type="button"
            onClick={handleSelectToggle}
            className={`w-5 h-5 rounded border-2 flex items-center justify-center transition-colors ${
              isSelected ? 'border-brand bg-brand/20' : 'border-caption bg-surface'
            }`}
            aria-label={isSelected ? 'Deselect clip' : 'Select clip'}
          >
            {isSelected && <span className="text-brand text-xs leading-none">✓</span>}
          </button>
        </div>
      </div>

      {/* Thumbnail + play overlay */}
      <div className="relative w-full aspect-video bg-background mx-0 mt-2 overflow-hidden">
        {thumbnailSrc ? (
          <img src={thumbnailSrc} alt={title || `Clip ${index + 1}`} className="w-full h-full object-cover" />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-caption text-xs">Sem thumbnail</div>
        )}
        {videoSrc && (
          <button
            type="button"
            onClick={handleVideoToggle}
            className="absolute inset-0 flex items-center justify-center bg-black/25 hover:bg-black/45 transition-colors"
            aria-label={videoOpen ? 'Fechar vídeo' : 'Assistir clip'}
          >
            <span className="w-12 h-12 rounded-full bg-white/15 border-2 border-white/50 flex items-center justify-center text-white text-lg">
              {videoOpen ? '✕' : '▶'}
            </span>
          </button>
        )}
      </div>

      {/* Video player (lazy — only in DOM when open) */}
      {videoOpen && videoSrc && (
        <div className="px-3 pt-2">
          <video
            ref={videoRef}
            src={videoSrc}
            controls
            autoPlay
            className="w-full rounded bg-black"
            style={{ maxHeight: '240px' }}
          />
        </div>
      )}

      {/* Editable fields */}
      <div className="p-3 space-y-2">
        <div>
          <div className="flex justify-between mb-0.5">
            <Label htmlFor={`clip-title-${clip.id}`} className="text-xs font-medium text-subtle">Título</Label>
            <span className="text-xs text-caption">{title.length}/100</span>
          </div>
          <Input
            id={`clip-title-${clip.id}`}
            type="text"
            value={title}
            onChange={handleTitleChange}
            placeholder="Título do clip..."
            maxLength={100}
            disabled={!isSelected}
          />
        </div>
        <div>
          <div className="flex justify-between mb-0.5">
            <Label htmlFor={`clip-thumb-${clip.id}`} className="text-xs font-medium text-subtle">Texto do Thumbnail</Label>
            <span className="text-xs text-caption">{thumbnailText.length}/30</span>
          </div>
          <Input
            id={`clip-thumb-${clip.id}`}
            type="text"
            value={thumbnailText}
            onChange={handleThumbnailTextChange}
            placeholder="GANCHO CURTO"
            maxLength={30}
            disabled={!isSelected}
            className="text-xs uppercase"
          />
        </div>
      </div>
    </div>
  );
}
