'use client';

import { useEffect, useState, useCallback, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { InfoBanner } from '@/components/ui/info-banner';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { useChannelStore } from '@/store/channelStore';
import { ClipReviewCard } from '@/components/pipeline/ClipReviewCard';
import type { ClipTextEdit, GatePayload, ModeConfig } from '@/types/pipeline';

interface StepReviewClipsProps {
  historical?: GatePayload;
}

export function StepReviewClips({ historical }: StepReviewClipsProps) {
  const isHistorical = historical !== undefined;

  const goUrl = useAppStore((s) => s.goUrl);
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const run = usePipelineStore((s) => s.run);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const submitReviewClips = usePipelineStore((s) => s.submitReviewClips);
  const resubmitFromBacked = usePipelineStore((s) => s.resubmitFromBacked);
  const channels = useChannelStore((s) => s.channels);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [titles, setTitles] = useState<Record<number, string>>({});
  const [thumbTexts, setThumbTexts] = useState<Record<number, string>>({});
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Parse upload_options from run.mode_config_json
  const uploadOptions = useMemo<ModeConfig['upload_options'] | undefined>(() => {
    if (!run?.mode_config_json) return undefined;
    try { return (JSON.parse(run.mode_config_json) as ModeConfig).upload_options; }
    catch { return undefined; }
  }, [run?.mode_config_json]);

  const buttonLabel = useMemo(() => {
    if (uploadOptions?.schedule_enabled) return 'Salvar na Fila';
    if (uploadOptions?.auto_enabled) return 'Fazer Upload';
    return 'Continuar';
  }, [uploadOptions]);

  const channel = useMemo(
    () => channels.find((c) => c.ID === run?.channel_id) ?? null,
    [channels, run?.channel_id],
  );

  // Load clips + channels on mount
  useEffect(() => {
    if (activeRunId == null) return;
    setLoading(true);
    loadClips(goUrl, activeRunId)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : 'Falha ao carregar clips');
      })
      .finally(() => setLoading(false));
  }, [goUrl, activeRunId]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    void fetchChannels(goUrl);
  }, [goUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  // Initialize local state from loaded clips
  useEffect(() => {
    if (clips.length === 0) return;
    const initTitles: Record<number, string> = {};
    const initTexts: Record<number, string> = {};

    for (const c of clips) {
      initTitles[c.id] = c.title ?? '';
      initTexts[c.id] = c.thumbnail_text ?? '';
    }

    // Pre-fill from historical payload (back-nav)
    if (isHistorical && historical?.kind === 'clips') {
      setSelectedIds(new Set(historical.selected_ids));
      if (historical.clip_edits) {
        for (const e of historical.clip_edits) {
          initTitles[e.id] = e.title;
          initTexts[e.id] = e.thumbnail_text;
        }
      }
    } else {
      const initSel = new Set<number>();
      for (const c of clips) {
        if (c.is_selected) initSel.add(c.id);
      }
      setSelectedIds(initSel);
    }

    setTitles(initTitles);
    setThumbTexts(initTexts);
  }, [clips]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSelectToggle = useCallback((id: number, sel: boolean) => {
    setSelectedIds((prev) => {
      const n = new Set(prev);
      sel ? n.add(id) : n.delete(id);
      return n;
    });
  }, []);

  const handleTitleChange = useCallback((id: number, v: string) => {
    setTitles((prev) => ({ ...prev, [id]: v }));
  }, []);

  const handleThumbTextChange = useCallback((id: number, v: string) => {
    setThumbTexts((prev) => ({ ...prev, [id]: v }));
  }, []);

  const handleSelectAll = useCallback(() => setSelectedIds(new Set(clips.map((c) => c.id))), [clips]);
  const handleDeselectAll = useCallback(() => setSelectedIds(new Set()), []);

  const handleSubmit = useCallback(async () => {
    if (!activeRunId) return;
    setSubmitting(true);
    setError(null);
    try {
      const clipEdits: ClipTextEdit[] = clips.map((c) => ({
        id: c.id,
        title: titles[c.id] ?? c.title,
        thumbnail_text: thumbTexts[c.id] ?? c.thumbnail_text,
      }));

      if (isHistorical) {
        await resubmitFromBacked(goUrl, 'WAITING_REVIEW_CLIPS', {
          kind: 'clips',
          selected_ids: Array.from(selectedIds),
          clip_edits: clipEdits,
        });
      } else {
        await submitReviewClips(goUrl, activeRunId, {
          selected_ids: Array.from(selectedIds),
          clip_edits: clipEdits,
        });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao salvar');
    } finally {
      setSubmitting(false);
    }
  }, [goUrl, activeRunId, clips, selectedIds, titles, thumbTexts, isHistorical, resubmitFromBacked, submitReviewClips]);

  if (loading) {
    return (
      <div className="space-y-2 max-w-lg">
        <h2 className="text-xl font-semibold text-heading">Revisar Clips</h2>
        <p className="text-sm text-subtle">Carregando clips...</p>
      </div>
    );
  }

  if (clips.length === 0) {
    return (
      <div className="space-y-2 max-w-lg">
        <h2 className="text-xl font-semibold text-heading">Revisar Clips</h2>
        <p className="text-sm text-subtle">Nenhum clip encontrado para este pipeline.</p>
      </div>
    );
  }

  const selectedCount = selectedIds.size;

  return (
    <div className="space-y-4 max-w-2xl">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-xl font-semibold text-heading">Revisar Clips</h2>
          <p className="text-sm text-subtle">Selecione os clips e ajuste título e texto do thumbnail.</p>
        </div>
        <span className="text-xs text-prose mt-1">{selectedCount}/{clips.length} selecionados</span>
      </div>

      {/* Historical notice */}
      {isHistorical && (
        <InfoBanner>
          Revisando submissão anterior. Re-submeter reinicia o pipeline a partir deste passo.
        </InfoBanner>
      )}

      {/* Upload config summary */}
      {uploadOptions && (
        <div className="rounded-md border border-border bg-card/50 px-4 py-3 text-xs space-y-1">
          <p className="text-prose font-medium mb-1">Configuração de upload</p>
          {channel && (
            <p className="text-subtle">
              Canal: <span className="text-prose">{channel.ChannelTitle || channel.Name}</span>
            </p>
          )}
          <p className="text-subtle">
            Privacidade: <span className="text-prose capitalize">{uploadOptions.privacy}</span>
          </p>
          {uploadOptions.schedule_enabled && (
            <p className="text-warning">Agendamento ativado</p>
          )}
          {uploadOptions.auto_enabled && (
            <p className="text-brand">Upload automático</p>
          )}
          {uploadOptions.dry_run && (
            <p className="text-warning">Dry run — nenhum upload será feito</p>
          )}
        </div>
      )}

      {/* Batch controls */}
      <div className="flex gap-3 text-xs">
        <button
          type="button"
          onClick={handleSelectAll}
          className="text-prose hover:text-heading underline underline-offset-2"
        >
          Selecionar todos
        </button>
        <span className="text-border">·</span>
        <button
          type="button"
          onClick={handleDeselectAll}
          className="text-prose hover:text-heading underline underline-offset-2"
        >
          Desmarcar todos
        </button>
      </div>

      {/* Error banner */}
      {error && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-xs text-destructive">
          {error}
        </div>
      )}

      {/* Clip cards */}
      <div className="space-y-4">
        {clips.map((clip, idx) => (
          <ClipReviewCard
            key={clip.id}
            clip={clip}
            index={idx}
            isSelected={selectedIds.has(clip.id)}
            title={titles[clip.id] ?? clip.title}
            thumbnailText={thumbTexts[clip.id] ?? clip.thumbnail_text}
            onSelectToggle={handleSelectToggle}
            onTitleChange={handleTitleChange}
            onThumbnailTextChange={handleThumbTextChange}
          />
        ))}
      </div>

      {/* Submit */}
      <div className="flex justify-end pt-2">
        <Button
          variant="brand"
          onClick={() => void handleSubmit()}
          disabled={submitting || selectedCount === 0}
        >
          {submitting ? 'Salvando…' : buttonLabel}
        </Button>
      </div>
    </div>
  );
}
