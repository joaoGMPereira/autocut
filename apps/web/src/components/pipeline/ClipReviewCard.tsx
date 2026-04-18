'use client';

import { useState, useRef, useCallback } from 'react';
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

  const formatDuration = (sec: number) => {
    const m = Math.floor(sec / 60);
    const s = Math.floor(sec % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  const handleVideoToggle = useCallback(() => {
    setVideoOpen((prev) => {
      if (prev && videoRef.current) videoRef.current.pause();
      return !prev;
    });
  }, []);

  return (
    <div
      className={`rounded-lg border transition-colors ${
        isSelected ? 'border-cyan-500/40 bg-cyan-500/5' : 'border-border bg-card/60 opacity-60'
      }`}
    >
      {/* Header: index, duration, selection */}
      <div className="flex items-center justify-between px-3 pt-3">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-zinc-400">Clip {index + 1}</span>
          <span className="text-xs text-zinc-600">{formatDuration(clip.duration_sec)}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-zinc-500">{isSelected ? 'Selecionado' : 'Ignorar'}</span>
          <button
            type="button"
            onClick={() => onSelectToggle(clip.id, !isSelected)}
            className={`w-5 h-5 rounded border-2 flex items-center justify-center transition-colors ${
              isSelected ? 'border-cyan-400 bg-cyan-400/20' : 'border-zinc-600 bg-zinc-800'
            }`}
            aria-label={isSelected ? 'Deselect clip' : 'Select clip'}
          >
            {isSelected && <span className="text-cyan-400 text-xs leading-none">✓</span>}
          </button>
        </div>
      </div>

      {/* Thumbnail + play overlay */}
      <div className="relative w-full aspect-video bg-zinc-900 mx-0 mt-2 overflow-hidden">
        {thumbnailSrc ? (
          <img src={thumbnailSrc} alt={title || `Clip ${index + 1}`} className="w-full h-full object-cover" />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-zinc-600 text-xs">Sem thumbnail</div>
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
            <label className="text-xs text-zinc-400">Título</label>
            <span className="text-xs text-zinc-600">{title.length}/100</span>
          </div>
          <input
            type="text"
            value={title}
            onChange={(e) => onTitleChange(clip.id, e.target.value)}
            placeholder="Título do clip..."
            maxLength={100}
            disabled={!isSelected}
            className="w-full text-sm bg-zinc-800 border border-border rounded px-2 py-1 text-foreground placeholder-zinc-600 focus:outline-none focus:border-cyan-500 disabled:opacity-40"
          />
        </div>
        <div>
          <div className="flex justify-between mb-0.5">
            <label className="text-xs text-zinc-400">Texto do Thumbnail</label>
            <span className="text-xs text-zinc-600">{thumbnailText.length}/30</span>
          </div>
          <input
            type="text"
            value={thumbnailText}
            onChange={(e) => onThumbnailTextChange(clip.id, e.target.value)}
            placeholder="GANCHO CURTO"
            maxLength={30}
            disabled={!isSelected}
            className="w-full text-xs bg-zinc-800 border border-border rounded px-2 py-1 text-foreground placeholder-zinc-600 focus:outline-none focus:border-cyan-500 uppercase disabled:opacity-40"
          />
        </div>
      </div>
    </div>
  );
}
