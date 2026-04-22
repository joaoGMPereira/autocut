'use client';

import { useRef, useState } from 'react';
import { Loader2, Trash2, Star } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useThumbnailStore } from '@/store/thumbnailStore';

interface BackgroundsGalleryProps {
  channelId: number;
}

export function BackgroundsGallery({ channelId }: BackgroundsGalleryProps) {
  const { backgrounds, selectedBgId, loadingBackgrounds, uploadBackground, deleteBackground, selectBackground } =
    useThumbnailStore();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadName, setUploadName] = useState('');
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const name = uploadName.trim() || file.name;
    setUploading(true);
    setUploadError(null);
    try {
      await uploadBackground(channelId, file, name);
      setUploadName('');
      if (fileInputRef.current) fileInputRef.current.value = '';
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (bgId: number) => {
    setDeletingId(bgId);
    try {
      await deleteBackground(channelId, bgId);
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      {/* Upload row */}
      <div className="flex items-center gap-2 flex-wrap">
        <input
          type="text"
          value={uploadName}
          onChange={(e) => setUploadName(e.target.value)}
          placeholder="Background name (optional)"
          className="rounded-lg bg-surface border border-border px-3 py-2 text-sm focus:outline-none focus:border-brand/60 w-52"
        />
        <Button
          variant="outline"
          size="sm"
          disabled={uploading}
          onClick={() => fileInputRef.current?.click()}
        >
          {uploading ? (
            <>
              <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              Uploading…
            </>
          ) : (
            'Upload Background'
          )}
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*,video/*"
          className="hidden"
          onChange={(e) => void handleFileChange(e)}
        />
        {uploadError && (
          <span className="text-xs text-destructive">{uploadError}</span>
        )}
      </div>

      {/* Gallery grid */}
      {loadingBackgrounds ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground py-4">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading backgrounds…
        </div>
      ) : backgrounds.length === 0 ? (
        <p className="text-sm text-subtle py-2">No backgrounds yet. Upload one above.</p>
      ) : (
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-4">
          {backgrounds.map((bg) => {
            const isSelected = selectedBgId === bg.id;
            return (
              <button
                key={bg.id}
                type="button"
                onClick={() => selectBackground(isSelected ? null : bg.id)}
                className={[
                  'group relative rounded-lg overflow-hidden border-2 transition-all focus:outline-none',
                  isSelected
                    ? 'border-brand ring-2 ring-brand/30'
                    : 'border-border hover:border-brand/40',
                ].join(' ')}
              >
                {/* Aspect-ratio wrapper */}
                <div className="aspect-video w-full bg-surface">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={bg.file_path}
                    alt={bg.name}
                    className="w-full h-full object-cover"
                  />
                </div>

                {/* Filename + default badge */}
                <div className="px-2 py-1 bg-surface-deep flex items-center gap-1 min-w-0">
                  {bg.is_default && (
                    <Star className="h-3 w-3 shrink-0 text-amber-400 fill-amber-400" />
                  )}
                  <span className="text-xs text-caption truncate">{bg.name}</span>
                </div>

                {/* Delete button (shown on hover) */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    void handleDelete(bg.id);
                  }}
                  disabled={deletingId === bg.id}
                  className="absolute top-1 right-1 hidden group-hover:flex items-center justify-center h-6 w-6 rounded bg-black/60 hover:bg-destructive/80 transition-colors"
                  aria-label="Delete background"
                >
                  {deletingId === bg.id ? (
                    <Loader2 className="h-3 w-3 animate-spin text-white" />
                  ) : (
                    <Trash2 className="h-3 w-3 text-white" />
                  )}
                </button>

                {/* Selected ring indicator */}
                {isSelected && (
                  <span className="absolute top-1 left-1 rounded bg-brand px-1.5 py-0.5 text-[10px] font-semibold text-black leading-none">
                    Selected
                  </span>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
