'use client';

import { useEffect, useState, useCallback } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { createLogger } from '@/lib/logger';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import type { GatePayload, Clip, ClipMetadataUpdate } from '@/types/pipeline';

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

export function StepReviewMetadata({ historical }: StepReviewMetadataProps) {
  const goUrl = useAppStore((s) => s.goUrl);
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const submitReviewMetadata = usePipelineStore((s) => s.submitReviewMetadata);
  const metadataProgress = usePipelineStore((s) => s.metadataProgress);

  const isHistorical = historical !== undefined;

  const [edits, setEdits] = useState<Record<number, ClipEdit>>({});
  const [isGenerating, setIsGenerating] = useState(false);
  const [generateError, setGenerateError] = useState<string | null>(null);
  const [regeneratingClipId, setRegeneratingClipId] = useState<number | null>(null);
  const [hasGenerated, setHasGenerated] = useState(false);

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

  // Auto-generate on mount (once, if clips have no metadata)
  useEffect(() => {
    if (!activeRunId || clips.length === 0 || hasGenerated || isHistorical) return;
    const hasMetadata = clips.some((c) => c.title !== '');
    if (!hasMetadata) {
      void handleGenerate();
    }
  }, [activeRunId, clips.length]); // eslint-disable-line react-hooks/exhaustive-deps

  // Watch for metadata_progress "done" to reload clips
  useEffect(() => {
    if (metadataProgress?.status === 'done' && activeRunId) {
      void loadClips(goUrl, activeRunId);
      setIsGenerating(false);
      setHasGenerated(true);
    }
    if (metadataProgress?.status === 'error') {
      setIsGenerating(false);
      setGenerateError(metadataProgress.message || 'Generation failed');
    }
  }, [metadataProgress?.status]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleGenerate = useCallback(async () => {
    if (!activeRunId) return;
    setIsGenerating(true);
    setGenerateError(null);
    try {
      const res = await fetch(`${goUrl}/api/metadata/runs/${activeRunId}/generate`, {
        method: 'POST',
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        throw new Error(data.error || `Failed: ${res.status}`);
      }
      // SSE events will update progress
    } catch (err) {
      setIsGenerating(false);
      setGenerateError(err instanceof Error ? err.message : 'Generation failed');
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
      // Update local edits with generated data
      setEdits((prev) => ({
        ...prev,
        [clipId]: {
          title: data.title || prev[clipId]?.title || '',
          description: data.description || prev[clipId]?.description || '',
          tags: Array.isArray(data.tags) ? data.tags.join(',') : (data.tags || prev[clipId]?.tags || ''),
          thumbnail_text: data.thumbnail_text || prev[clipId]?.thumbnail_text || '',
        },
      }));
    } catch (err) {
      log.error('Regenerate single failed', { clipId, error: err instanceof Error ? err.message : 'Unknown' });
    } finally {
      setRegeneratingClipId(null);
    }
  }, [goUrl, activeRunId]);

  const updateEdit = useCallback((clipId: number, field: keyof ClipEdit, value: string) => {
    setEdits((prev) => ({
      ...prev,
      [clipId]: { ...prev[clipId], [field]: value },
    }));
  }, []);

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

  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-foreground">Review Metadata</h2>
          <p className="text-sm text-zinc-500">
            Edit titles, descriptions, tags, and thumbnail text for each clip.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void handleGenerate()}
          disabled={isGenerating || clips.length === 0}
        >
          {isGenerating ? 'Generating...' : 'Generate All'}
        </Button>
      </div>

      {isHistorical && (
        <div className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3 text-xs text-zinc-400">
          Reviewing previous submission. Re-submitting will restart the pipeline from this step.
        </div>
      )}

      {/* Progress bar during generation */}
      {isGenerating && metadataProgress && (
        <div className="space-y-1">
          <Progress value={metadataProgress.percent} className="h-2" />
          <p className="text-xs text-zinc-500">{metadataProgress.message}</p>
        </div>
      )}

      {/* Error banner */}
      {generateError && (
        <div className="rounded-md border border-red-800 bg-red-950 px-4 py-3 text-xs text-red-400">
          {generateError}
        </div>
      )}

      {/* Clip cards */}
      <div className="space-y-4">
        {clips.map((clip, idx) => {
          const edit = edits[clip.id];
          if (!edit) return null;
          const isRegen = regeneratingClipId === clip.id;

          return (
            <div
              key={clip.id}
              className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-4 space-y-3"
            >
              {/* Card header */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary" className="text-xs">
                    Clip {idx + 1}
                  </Badge>
                  <span className="text-xs text-zinc-500">
                    {formatDuration(clip.start_sec)} – {formatDuration(clip.end_sec)} ({formatDuration(clip.duration_sec)})
                  </span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => void handleRegenerateSingle(clip.id)}
                  disabled={isRegen || isGenerating}
                  className="text-xs"
                >
                  {isRegen ? 'Regenerating...' : 'Regenerate'}
                </Button>
              </div>

              {/* Thumbnail text */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs text-zinc-400">Thumbnail Text</Label>
                  <span className="text-xs text-zinc-600">{edit.thumbnail_text.length}/30</span>
                </div>
                <Input
                  value={edit.thumbnail_text}
                  onChange={(e) => updateEdit(clip.id, 'thumbnail_text', e.target.value)}
                  placeholder="SHORT HOOK (2-3 words, ALL CAPS)"
                  maxLength={30}
                  className="text-sm uppercase"
                />
              </div>

              {/* Title */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs text-zinc-400">Title</Label>
                  <span className="text-xs text-zinc-600">{edit.title.length}/100</span>
                </div>
                <Input
                  value={edit.title}
                  onChange={(e) => updateEdit(clip.id, 'title', e.target.value)}
                  placeholder="Clip title..."
                  maxLength={100}
                  className="text-sm"
                />
              </div>

              {/* Description */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs text-zinc-400">Description</Label>
                  <span className="text-xs text-zinc-600">{edit.description.length}/5000</span>
                </div>
                <textarea
                  value={edit.description}
                  onChange={(e) => updateEdit(clip.id, 'description', e.target.value)}
                  placeholder="Clip description..."
                  maxLength={5000}
                  rows={3}
                  className="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm text-foreground placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-600"
                />
              </div>

              {/* Tags */}
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <Label className="text-xs text-zinc-400">Tags</Label>
                  <span className="text-xs text-zinc-600">{edit.tags.length}/500</span>
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
          <Button onClick={() => void handleSubmit()} disabled={isGenerating}>
            Continue
          </Button>
        </div>
      )}
    </div>
  );
}
