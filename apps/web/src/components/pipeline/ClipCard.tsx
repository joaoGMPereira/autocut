'use client';

import { useState } from 'react';
import type { Clip } from '@/types/pipeline';

type ClipCardVariant = 'metadata' | 'select';

interface ClipCardProps {
  clip: Clip;
  variant: ClipCardVariant;
  onMetadataChange?: (id: number, title: string, description: string, tags: string) => void;
  onSelectToggle?: (id: number, selected: boolean) => void;
}

export function ClipCard({ clip, variant, onMetadataChange, onSelectToggle }: ClipCardProps) {
  const [title, setTitle] = useState(clip.title);
  const [description, setDescription] = useState(clip.description);
  const [tags, setTags] = useState(clip.tags);
  const [selected, setSelected] = useState(clip.is_selected);

  function emitMetadata(t: string, d: string, tg: string) {
    onMetadataChange?.(clip.id, t, d, tg);
  }

  function handleToggle() {
    const next = !selected;
    setSelected(next);
    onSelectToggle?.(clip.id, next);
  }

  return (
    <div className={`rounded-lg border p-3 space-y-2 transition-colors ${selected ? 'border-brand/40 bg-brand/5' : 'border-border bg-card/60'}`}>
      {/* Thumbnail */}
      <div className="w-full aspect-video bg-zinc-800 rounded overflow-hidden flex items-center justify-center relative">
        {clip.thumbnail_path ? (
          <img
            src={`/files/${encodeURIComponent(clip.thumbnail_path)}`}
            alt={clip.title || `Clip ${clip.id}`}
            className="w-full h-full object-cover"
          />
        ) : (
          <span className="text-zinc-600 text-xs">No thumbnail</span>
        )}
        {variant === 'select' && (
          <button
            type="button"
            onClick={handleToggle}
            className={`absolute top-2 right-2 w-6 h-6 rounded border-2 flex items-center justify-center transition-colors ${selected ? 'border-brand bg-brand/30' : 'border-zinc-500 bg-black/40'}`}
            aria-label={selected ? 'Deselect clip' : 'Select clip'}
          >
            {selected && <span className="text-brand text-xs font-bold">✓</span>}
          </button>
        )}
      </div>

      {/* Duration */}
      <div className="flex items-center justify-between text-xs text-zinc-500">
        <span>{clip.duration_sec.toFixed(1)}s</span>
        {clip.upload_status && clip.upload_status !== 'pending' && (
          <span className={`font-medium ${clip.upload_status === 'done' ? 'text-emerald-400' : clip.upload_status === 'error' ? 'text-red-400' : 'text-amber-400'}`}>
            {clip.upload_status}
          </span>
        )}
      </div>

      {/* Metadata editor — only in metadata variant */}
      {variant === 'metadata' && (
        <div className="space-y-2">
          <input
            type="text"
            value={title}
            placeholder="Title"
            onChange={(e) => { setTitle(e.target.value); emitMetadata(e.target.value, description, tags); }}
            className="w-full text-sm bg-zinc-800 border border-border rounded px-2 py-1 text-foreground placeholder-zinc-500 focus:outline-none focus:border-brand"
          />
          <textarea
            value={description}
            placeholder="Description"
            rows={2}
            onChange={(e) => { setDescription(e.target.value); emitMetadata(title, e.target.value, tags); }}
            className="w-full text-xs bg-zinc-800 border border-border rounded px-2 py-1 text-foreground placeholder-zinc-500 focus:outline-none focus:border-brand resize-none"
          />
          <input
            type="text"
            value={tags}
            placeholder="Tags (comma-separated)"
            onChange={(e) => { setTags(e.target.value); emitMetadata(title, description, e.target.value); }}
            className="w-full text-xs bg-zinc-800 border border-border rounded px-2 py-1 text-foreground placeholder-zinc-500 focus:outline-none focus:border-brand"
          />
        </div>
      )}

      {/* YouTube link — shown when done */}
      {clip.youtube_id && (
        <a
          href={`https://www.youtube.com/watch?v=${clip.youtube_id}`}
          target="_blank"
          rel="noopener noreferrer"
          className="block text-xs text-brand hover:underline truncate"
        >
          youtube.com/watch?v={clip.youtube_id}
        </a>
      )}
    </div>
  );
}
