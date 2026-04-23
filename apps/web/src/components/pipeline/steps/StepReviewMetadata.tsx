'use client';

import { useEffect, useState, useCallback, useRef } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { createLogger } from '@/lib/logger';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import type { GatePayload, ClipMetadataUpdate } from '@/types/pipeline';

const log = createLogger('StepReviewMetadata');

interface StepReviewMetadataProps {
  historical?: GatePayload;
}

interface ClipEdit {
  title: string;
  description: string;
  tags: string;
  thumbnail_text: string;
}

function phaseLabel(status: string | undefined): string {
  switch (status) {
    case 'fetching_channel': return 'Baixando dados do canal...';
    case 'generating':       return 'Preparando geração...';
    case 'streaming':        return 'Gerando metadados com IA...';
    default:                 return 'Aguardando início...';
  }
}

export function StepReviewMetadata({ historical }: StepReviewMetadataProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const submitReviewMetadata = usePipelineStore((s) => s.submitReviewMetadata);
  const metadataProgress = usePipelineStore((s) => s.metadataProgress);

  const isHistorical = historical !== undefined;

  const [edits, setEdits] = useState<Record<number, ClipEdit>>({});
  const [generateError, setGenerateError] = useState<string | null>(null);
  const [regeneratingClipId, setRegeneratingClipId] = useState<number | null>(null);

  // Thumbnail preview state
  const [thumbCacheBust, setThumbCacheBust] = useState<Record<number, number>>({});
  const [thumbRegen, setThumbRegen] = useState<Set<number>>(new Set());
  const thumbDebounceRef = useRef<Record<number, ReturnType<typeof setTimeout>>>({});
  // Set to true after AI metadata generation completes, triggers auto-regen
  const [autoRegenPending, setAutoRegenPending] = useState(false);
  // Blocks the editor until the initial thumbnail regen batch is done.
  // Starts true — only flips false when we actually kick off a regen batch.
  const [thumbsReady, setThumbsReady] = useState(true);
  // True only after at least one successful regen batch has completed.
  const [thumbsHaveBeenGenerated, setThumbsHaveBeenGenerated] = useState(false);

  // True while the backend is auto-generating (PrefillBatch goroutine or manual re-trigger).
  // description starts empty and is only set after AI runs — safer sentinel than title
  // (clips always have a default "Part N" title before AI generation)
  const hasMetadata = clips.length > 0 && clips.some((c) => c.description !== '' || c.tags !== '');
  const isActivelyGenerating =
    metadataProgress?.status === 'fetching_channel' ||
    metadataProgress?.status === 'generating' ||
    metadataProgress?.status === 'streaming';

  // While clips haven't loaded yet we show nothing — avoid flicker between loading and editor.
  const clipsLoaded = clips.length > 0;
  // Show loading screen when: metadata generating OR (metadata exists but thumbnails not ready).
  const showLoading = !isHistorical && (
    isActivelyGenerating ||
    (clipsLoaded && !hasMetadata && metadataProgress?.status !== 'error') ||
    (clipsLoaded && hasMetadata && !thumbsReady)
  );

  // ── Thumbnail preview regeneration ──────────────────────────────────────────
  const regenThumbnail = useCallback(async (clipId: number, text: string) => {
    if (!activeRunId) return;
    setThumbRegen((prev) => new Set(prev).add(clipId));
    try {
      const res = await fetch(
        `${goUrl}/api/thumbnail/runs/${activeRunId}/clips/${clipId}/preview-text`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text }),
        }
      );
      if (res.ok || res.status === 204) {
        setThumbCacheBust((prev) => ({ ...prev, [clipId]: Date.now() }));
      }
    } catch (err) {
      log.error('thumbnail preview-text failed', { clipId, error: err instanceof Error ? err.message : 'unknown' });
    } finally {
      setThumbRegen((prev) => { const s = new Set(prev); s.delete(clipId); return s; });
    }
  }, [goUrl, activeRunId]);

  // Auto-regen all thumbnails that have thumbnail_text — fires on initial load
  // (once activeRunId is known) and whenever AI metadata generation completes.
  // Blocks the editor (thumbsReady=false) until all regens finish.
  const initialRegenDoneRef = useRef<number | null>(null); // stores runId of last regen
  useEffect(() => {
    if (!activeRunId || clips.length === 0) return;
    const shouldRegen = autoRegenPending || initialRegenDoneRef.current !== activeRunId;
    if (!shouldRegen) return;

    const clipsWithText = clips.filter((c) => !!c.thumbnail_text);
    // No clips with text yet (metadata still generating or loadClips not done) — wait.
    // Don't consume the flag here so we retry when clips reload with thumbnail_text.
    if (clipsWithText.length === 0) return;

    // Consume flags only after confirming there are clips to regen.
    initialRegenDoneRef.current = activeRunId;
    setAutoRegenPending(false);

    setThumbsReady(false);
    void Promise.all(clipsWithText.map((clip) => regenThumbnail(clip.id, clip.thumbnail_text)))
      .finally(() => {
        setThumbsReady(true);
        setThumbsHaveBeenGenerated(true);
      });
  }, [activeRunId, autoRegenPending, clips, regenThumbnail]);

  // Load clips on mount
  useEffect(() => {
    if (!activeRunId) return;
    void loadClips(goUrl, activeRunId);
  }, [goUrl, activeRunId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Initialize edits from clips
  useEffect(() => {
    const newEdits: Record<number, ClipEdit> = {};
    for (const clip of clips) {
      newEdits[clip.id] = {
        title: clip.title || '',
        description: clip.description || '',
        tags: clip.tags || '',
        thumbnail_text: clip.thumbnail_text || '',
      };
    }
    setEdits(newEdits);
  }, [clips]);

  // Watch for metadata_progress events
  useEffect(() => {
    if (metadataProgress?.status === 'done' && activeRunId) {
      void loadClips(goUrl, activeRunId);
      setAutoRegenPending(true);
    }
    if (metadataProgress?.status === 'error') {
      setGenerateError(metadataProgress.message || 'Geração falhou');
    }
  }, [metadataProgress?.status]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleRegenerate = useCallback(async () => {
    if (!activeRunId) return;
    setGenerateError(null);
    try {
      const res = await fetch(`${goUrl}/api/metadata/runs/${activeRunId}/generate`, {
        method: 'POST',
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        throw new Error(data.error || `Failed: ${res.status}`);
      }
    } catch (err) {
      setGenerateError(err instanceof Error ? err.message : 'Geração falhou');
    }
  }, [goUrl, activeRunId]);

  const handleRegenerateSingle = useCallback(async (clipId: number) => {
    if (!activeRunId) return;
    setRegeneratingClipId(clipId);
    try {
      const res = await fetch(
        `${goUrl}/api/metadata/runs/${activeRunId}/clips/${clipId}/generate`,
        { method: 'POST' }
      );
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = await res.json();
      const newText = Array.isArray(data.tags) ? data.tags.join(',') : (data.tags || '');
      setEdits((prev) => ({
        ...prev,
        [clipId]: {
          title: data.title || prev[clipId]?.title || '',
          description: data.description || prev[clipId]?.description || '',
          tags: newText || prev[clipId]?.tags || '',
          thumbnail_text: data.thumbnail_text || prev[clipId]?.thumbnail_text || '',
        },
      }));
      // Regenerate thumbnail with new text
      const newThumbText = data.thumbnail_text || '';
      if (newThumbText) {
        void regenThumbnail(clipId, newThumbText);
      }
    } catch (err) {
      log.error('Regenerate single failed', { clipId, error: err instanceof Error ? err.message : 'Unknown' });
    } finally {
      setRegeneratingClipId(null);
    }
  }, [goUrl, activeRunId, regenThumbnail]);

  const updateEdit = useCallback((clipId: number, field: keyof ClipEdit, value: string) => {
    setEdits((prev) => ({
      ...prev,
      [clipId]: { ...prev[clipId], [field]: value },
    }));
    if (field === 'thumbnail_text') {
      clearTimeout(thumbDebounceRef.current[clipId]);
      thumbDebounceRef.current[clipId] = setTimeout(() => {
        void regenThumbnail(clipId, value);
      }, 700);
    }
  }, [regenThumbnail]);

  const handleSubmit = useCallback(async () => {
    if (!activeRunId) return;
    const clipEdits: ClipMetadataUpdate[] = Object.entries(edits).map(([id, edit]) => ({
      id: parseInt(id, 10),
      title: edit.title,
      description: edit.description,
      tags: edit.tags,
      thumbnail_text: edit.thumbnail_text,
    }));
    await submitReviewMetadata(goUrl, activeRunId, { clips: clipEdits });
  }, [goUrl, activeRunId, edits, submitReviewMetadata]);

  const formatDuration = (sec: number) => {
    const m = Math.floor(sec / 60);
    const s = Math.floor(sec % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  // ── Loading screen (auto-generation in progress) ──────────────────────────
  if (!isHistorical && !clipsLoaded) {
    return (
      <div className="flex items-center justify-center min-h-[300px]">
        <p className="text-sm text-subtle">Carregando clips...</p>
      </div>
    );
  }

  if (showLoading) {
    const status = metadataProgress?.status;
    const percent = metadataProgress?.percent ?? 0;

    return (
      <div className="max-w-2xl space-y-8">
        <div>
          <h2 className="text-xl font-semibold text-foreground">Preparando metadados</h2>
          <p className="text-sm text-subtle mt-1">
            Os metadados estão sendo gerados automaticamente. Aguarde...
          </p>
        </div>

        <div className="rounded-lg border border-border bg-card/50 p-6 space-y-4">
          {/* Phase indicator */}
          <div className="flex items-center gap-3">
            <div className="h-2 w-2 rounded-full bg-primary animate-pulse" />
            <span className="text-sm font-medium text-foreground">
              {phaseLabel(status)}
            </span>
          </div>

          {/* Progress bar */}
          <Progress value={percent} className="h-2" />

          {/* Step messages */}
          <div className="space-y-2">
            <StepRow
              done={status === 'generating' || status === 'streaming' || status === 'done'}
              active={status === 'fetching_channel'}
              label="Baixar dados do canal"
            />
            <StepRow
              done={!isActivelyGenerating && hasMetadata}
              active={status === 'generating' || status === 'streaming'}
              label="Gerar metadados com IA"
            />
            <StepRow
              done={thumbsHaveBeenGenerated && thumbsReady}
              active={hasMetadata && !isActivelyGenerating && !thumbsReady}
              label="Gerar thumbnails com texto"
            />
          </div>

          {/* Thumbnail previews while regenerating */}
          {status === 'done' && !thumbsReady && clips.length > 0 && (
            <div className="flex gap-2 pt-1">
              {clips.map((clip) => (
                <div
                  key={clip.id}
                  className="flex-1 rounded overflow-hidden bg-surface aspect-video relative"
                >
                  <img
                    key={thumbCacheBust[clip.id] ?? 0}
                    src={`${goUrl}/api/thumbnail/runs/${activeRunId}/clips/${clip.id}/file?t=${thumbCacheBust[clip.id] ?? 0}`}
                    alt=""
                    className={`w-full h-full object-cover transition-opacity duration-300 ${thumbRegen.has(clip.id) ? 'opacity-40' : 'opacity-100'}`}
                    onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none'; }}
                  />
                  {thumbRegen.has(clip.id) && (
                    <div className="absolute inset-0 flex items-center justify-center">
                      <div className="h-1.5 w-1.5 rounded-full bg-primary animate-pulse" />
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {metadataProgress?.message && (
            <p className="text-xs text-subtle">{metadataProgress.message}</p>
          )}
        </div>
      </div>
    );
  }

  // ── Editor ────────────────────────────────────────────────────────────────
  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-foreground">Revisar metadados</h2>
          <p className="text-sm text-subtle">
            Edite títulos, descrições, tags e texto de thumbnail de cada clip.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void handleRegenerate()}
          disabled={isActivelyGenerating || clips.length === 0}
        >
          {isActivelyGenerating ? 'Gerando...' : 'Regenerar tudo'}
        </Button>
      </div>

      {isHistorical && (
        <div className="rounded-md border border-border bg-card px-4 py-3 text-xs text-subtle">
          Revisando submissão anterior. Re-submeter reinicia o pipeline a partir deste passo.
        </div>
      )}

      {/* Progress bar during manual re-generation */}
      {isActivelyGenerating && metadataProgress && (
        <div className="space-y-1">
          <Progress value={metadataProgress.percent} className="h-2" />
          <p className="text-xs text-subtle">{metadataProgress.message}</p>
        </div>
      )}

      {/* Error banner */}
      {generateError && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-xs text-destructive">
          {generateError}
        </div>
      )}

      {/* Clip cards */}
      <div className="space-y-4">
        {clips.map((clip, idx) => {
          const edit = edits[clip.id];
          if (!edit) return null;
          const isRegen = regeneratingClipId === clip.id;
          const isThumbRegen = thumbRegen.has(clip.id);
          const cacheBust = thumbCacheBust[clip.id] ?? 0;

          return (
            <div
              key={clip.id}
              className="rounded-lg border border-border bg-card/50 p-4 space-y-3"
            >
              {/* Thumbnail preview */}
              <div
                className={`relative rounded-lg overflow-hidden bg-surface ${
                  clip.thumbnail_style === 'landscape'
                    ? 'w-full aspect-video'
                    : 'mx-auto aspect-[9/16] w-44'
                }`}
              >
                {isThumbRegen && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/60 z-10">
                    <span className="text-xs text-prose animate-pulse">Atualizando thumbnail...</span>
                  </div>
                )}
                <img
                  key={cacheBust}
                  src={`${goUrl}/api/thumbnail/runs/${activeRunId}/clips/${clip.id}/file?t=${cacheBust}`}
                  alt="thumbnail"
                  className="w-full h-full object-cover"
                  onError={(e) => {
                    (e.currentTarget as HTMLImageElement).style.display = 'none';
                  }}
                />
              </div>

              {/* Card header */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary" className="text-xs">
                    Clip {idx + 1}
                  </Badge>
                  <ThumbnailTypeBadge style={clip.thumbnail_style} />
                  <span className="text-xs text-subtle">
                    {formatDuration(clip.start_sec)} – {formatDuration(clip.end_sec)} ({formatDuration(clip.duration_sec)})
                  </span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => void handleRegenerateSingle(clip.id)}
                  disabled={isRegen || isActivelyGenerating}
                  className="text-xs"
                >
                  {isRegen ? 'Regenerando...' : 'Regenerar'}
                </Button>
              </div>

              {/* Thumbnail text */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs font-medium text-subtle">Thumbnail Text</Label>
                  <span className="text-xs text-caption">{edit.thumbnail_text.length}/30</span>
                </div>
                <Input
                  value={edit.thumbnail_text}
                  onChange={(e) => updateEdit(clip.id, 'thumbnail_text', e.target.value)}
                  placeholder="GANCHO CURTO (2-3 palavras, CAIXA ALTA)"
                  maxLength={30}
                  className="text-sm uppercase"
                />
              </div>

              {/* Title */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs font-medium text-subtle">Título</Label>
                  <span className="text-xs text-caption">{edit.title.length}/100</span>
                </div>
                <Input
                  value={edit.title}
                  onChange={(e) => updateEdit(clip.id, 'title', e.target.value)}
                  placeholder="Título do clip..."
                  maxLength={100}
                  className="text-sm"
                />
              </div>

              {/* Description */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs font-medium text-subtle">Descrição</Label>
                  <span className="text-xs text-caption">{edit.description.length}/5000</span>
                </div>
                <textarea
                  value={edit.description}
                  onChange={(e) => updateEdit(clip.id, 'description', e.target.value)}
                  placeholder="Descrição do clip..."
                  maxLength={5000}
                  rows={3}
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-caption focus:outline-none focus-visible:border-brand/60 focus-visible:ring-brand/30 focus-visible:ring-[3px]"
                />
              </div>

              {/* Tags */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs font-medium text-subtle">Tags</Label>
                  <span className="text-xs text-caption">{edit.tags.length}/500</span>
                </div>
                <Input
                  value={edit.tags}
                  onChange={(e) => updateEdit(clip.id, 'tags', e.target.value)}
                  placeholder="tag1, tag2, tag3..."
                  maxLength={500}
                  className="text-sm"
                />
              </div>
            </div>
          );
        })}
      </div>

      {/* Submit button */}
      {clips.length > 0 && (
        <div className="flex justify-end pt-2">
          <Button onClick={() => void handleSubmit()} disabled={isActivelyGenerating}>
            Continuar
          </Button>
        </div>
      )}
    </div>
  );
}

// ── helpers ───────────────────────────────────────────────────────────────────

function StepRow({ done, active, label }: { done: boolean; active: boolean; label: string }) {
  return (
    <div className="flex items-center gap-2 text-xs">
      {done ? (
        <span className="text-success">✓</span>
      ) : active ? (
        <span className="h-2 w-2 rounded-full bg-primary animate-pulse inline-block" />
      ) : (
        <span className="h-2 w-2 rounded-full bg-border inline-block" />
      )}
      <span className={done ? 'text-subtle line-through' : active ? 'text-foreground' : 'text-caption'}>
        {label}
      </span>
    </div>
  );
}

function ThumbnailTypeBadge({ style }: { style: string }) {
  const isLandscape = style === 'landscape';
  return (
    <Badge
      variant={isLandscape ? 'warning' : 'secondary'}
      className={
        isLandscape
          ? 'text-[10px] font-semibold'
          : 'text-[10px] font-semibold border-violet/30 bg-violet/15 text-violet'
      }
    >
      {isLandscape ? 'Landscape' : 'Shorts'}
    </Badge>
  );
}
