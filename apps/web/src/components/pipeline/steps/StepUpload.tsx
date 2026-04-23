'use client';

import { useEffect, useMemo, useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { useChannelStore } from '@/store/channelStore';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { Clip, ModeConfig } from '@/types/pipeline';

// ── helpers ───────────────────────────────────────────────────────────────────

function fmtDuration(sec: number): string {
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = sec % 60;
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  return `${m}:${String(s).padStart(2, '0')}`;
}

// ── Privacy badge ─────────────────────────────────────────────────────────────

function PrivacyBadge({ privacy }: { privacy: string }) {
  const map: Record<string, { label: string; variant: 'secondary' | 'warning' | 'success' }> = {
    private:  { label: '🔒 Private',  variant: 'secondary' },
    unlisted: { label: '🔗 Unlisted', variant: 'warning' },
    public:   { label: '🌐 Public',   variant: 'success' },
  };
  const { label, variant } = map[privacy] ?? map['private'];
  return <Badge variant={variant}>{label}</Badge>;
}

// ── Screen 1: Confirmation ────────────────────────────────────────────────────

interface ConfirmScreenProps {
  clips: Clip[];
  uploadOptions?: ModeConfig['upload_options'];
  channelName?: string;
  channelAvatarUrl?: string;
  onConfirm: (mode: 'queue' | 'direct') => void;
  onSkip: () => void;
  submitting: boolean;
  error: string | null;
}

function ConfirmScreen({
  clips, uploadOptions, channelName, channelAvatarUrl,
  onConfirm, onSkip, submitting, error,
}: ConfirmScreenProps) {
  const selectedClips = clips.filter((c) => c.is_selected);
  const totalDurationSec = selectedClips.reduce((sum, c) => sum + (c.duration_sec ?? 0), 0);

  const defaultMode = uploadOptions?.mode ?? 'queue';
  const [mode, setMode] = useState<'queue' | 'direct'>(defaultMode);
  const privacy = uploadOptions?.privacy ?? 'private';

  return (
    <div className="space-y-5 max-w-lg">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold text-foreground">Confirmar Upload</h2>
        <p className="text-sm text-subtle">Revise as configurações antes de prosseguir.</p>
      </div>

      {/* Summary card */}
      <div className="rounded-lg border border-border bg-card divide-y divide-border">
        {/* Channel */}
        {channelName && (
          <div className="flex items-center gap-3 px-4 py-3">
            {channelAvatarUrl && (
              <img src={channelAvatarUrl} alt="" className="h-8 w-8 rounded-full object-cover flex-shrink-0" />
            )}
            <div className="min-w-0">
              <p className="text-xs text-subtle">Canal</p>
              <p className="text-sm font-medium text-foreground truncate">{channelName}</p>
            </div>
          </div>
        )}

        {/* Clip count & duration */}
        <div className="grid grid-cols-2 divide-x divide-border">
          <div className="px-4 py-3">
            <p className="text-xs text-subtle">Clips selecionados</p>
            <p className="text-lg font-semibold text-heading">{selectedClips.length}</p>
          </div>
          <div className="px-4 py-3">
            <p className="text-xs text-subtle">Duração total</p>
            <p className="text-lg font-semibold text-heading">{fmtDuration(totalDurationSec)}</p>
          </div>
        </div>

        {/* Privacy */}
        <div className="flex items-center justify-between px-4 py-3">
          <p className="text-xs text-subtle">Privacidade</p>
          <PrivacyBadge privacy={privacy} />
        </div>
      </div>

      {/* Mode selector */}
      <div className="space-y-2">
        <p className="text-xs font-medium text-subtle uppercase tracking-wider">Modo de Upload</p>
        <div className="grid grid-cols-2 gap-3">
          <button
            onClick={() => setMode('queue')}
            className={[
              'flex flex-col gap-1.5 rounded-xl border p-4 text-left transition-all',
              mode === 'queue'
                ? 'border-foreground bg-surface ring-1 ring-foreground'
                : 'border-border bg-card hover:border-border/80 hover:bg-surface/50',
            ].join(' ')}
          >
            {mode === 'queue' && (
              <span className="self-end flex h-4 w-4 items-center justify-center rounded-full bg-foreground text-background text-[10px] font-bold">✓</span>
            )}
            <span className="text-base">🗄️</span>
            <span className="text-sm font-semibold text-foreground">Salvar na Fila</span>
            <span className="text-xs text-subtle leading-relaxed">
              Salva os clips para upload posterior pela página de Queue.
            </span>
          </button>

          <button
            onClick={() => setMode('direct')}
            className={[
              'flex flex-col gap-1.5 rounded-xl border p-4 text-left transition-all',
              mode === 'direct'
                ? 'border-foreground bg-surface ring-1 ring-foreground'
                : 'border-border bg-card hover:border-border/80 hover:bg-surface/50',
            ].join(' ')}
          >
            {mode === 'direct' && (
              <span className="self-end flex h-4 w-4 items-center justify-center rounded-full bg-foreground text-background text-[10px] font-bold">✓</span>
            )}
            <span className="text-base">🚀</span>
            <span className="text-sm font-semibold text-foreground">Upload Direto</span>
            <span className="text-xs text-subtle leading-relaxed">
              Faz o upload agora diretamente para o YouTube com progresso em tempo real.
            </span>
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-xs text-destructive">
          {error}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between pt-1">
        <Button
          type="button"
          variant="link"
          onClick={onSkip}
          disabled={submitting}
          className="text-xs text-caption hover:text-subtle h-auto p-0"
        >
          Pular Upload
        </Button>

        <Button
          type="button"
          onClick={() => onConfirm(mode)}
          disabled={submitting || selectedClips.length === 0}
        >
          {submitting
            ? 'Processando…'
            : mode === 'direct'
              ? `Iniciar Upload (${selectedClips.length} clip${selectedClips.length !== 1 ? 's' : ''})`
              : `Salvar na Fila (${selectedClips.length} clip${selectedClips.length !== 1 ? 's' : ''})`}
        </Button>
      </div>
    </div>
  );
}

// ── Screen 2: Upload progress ─────────────────────────────────────────────────

interface ProgressScreenProps {
  clips: Clip[];
  clipProgress: Record<string, number> | null;
  onCancel: () => void;
}

function ProgressScreen({ clips, clipProgress, onCancel }: ProgressScreenProps) {
  const selectedClips = clips.filter((c) => c.is_selected);

  const uploadedCount = selectedClips.filter(
    (c) => c.upload_status === 'uploaded' || (clipProgress?.[c.id] ?? 0) >= 100
  ).length;

  const overallPct = selectedClips.length === 0
    ? 0
    : Math.round(
        selectedClips.reduce((sum, c) => sum + Math.min(clipProgress?.[c.id] ?? 0, 100), 0) /
        selectedClips.length
      );

  return (
    <div className="space-y-5 max-w-lg">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold text-foreground">Fazendo Upload</h2>
        <p className="text-sm text-subtle">
          {uploadedCount}/{selectedClips.length} clips enviados
        </p>
      </div>

      {/* Overall progress bar */}
      <div className="space-y-1.5">
        <div className="flex items-center justify-between text-xs text-subtle">
          <span>Progresso geral</span>
          <span>{overallPct}%</span>
        </div>
        <div className="h-2 w-full rounded-full bg-surface overflow-hidden">
          <div
            className="h-full rounded-full bg-foreground transition-all duration-300"
            style={{ width: `${overallPct}%` }}
          />
        </div>
      </div>

      {/* Per-clip progress */}
      <div className="space-y-3">
        {selectedClips.map((clip) => {
          const pct = Math.min(Math.round(clipProgress?.[clip.id] ?? 0), 100);
          const isDone = clip.upload_status === 'uploaded' || clip.youtube_id;
          const isError = clip.upload_status === 'error';

          return (
            <div
              key={clip.id}
              className="rounded-lg border border-border bg-card px-4 py-3 space-y-2"
            >
              <div className="flex items-start justify-between gap-2">
                <p className="text-xs font-medium text-prose truncate flex-1">{clip.title}</p>
                {isDone && clip.youtube_id && (
                  <a
                    href={`https://youtube.com/watch?v=${clip.youtube_id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex-shrink-0 text-xs text-brand hover:text-brand/80 underline underline-offset-2"
                  >
                    Ver no YT ↗
                  </a>
                )}
                {isError && (
                  <Badge variant="destructive" className="flex-shrink-0 text-xs">Erro</Badge>
                )}
              </div>

              {!isDone && !isError && (
                <>
                  <div className="h-1.5 w-full rounded-full bg-surface overflow-hidden">
                    <div
                      className="h-full rounded-full bg-subtle transition-all duration-200"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <div className="flex items-center justify-between text-[11px] text-caption">
                    <span>{pct}%</span>
                  </div>
                </>
              )}

              {isDone && (
                <div className="text-[11px] text-success font-medium">✓ Enviado</div>
              )}
            </div>
          );
        })}
      </div>

      <div className="flex justify-end">
        <Button
          type="button"
          variant="link"
          onClick={onCancel}
          className="text-xs text-caption hover:text-destructive h-auto p-0"
        >
          Cancelar upload
        </Button>
      </div>
    </div>
  );
}

// ── Screen 3: Done ────────────────────────────────────────────────────────────

interface DoneScreenProps {
  clips: Clip[];
  mode: 'queue' | 'direct' | null;
  onNewPipeline: () => void;
}

function DoneScreen({ clips, mode, onNewPipeline }: DoneScreenProps) {
  const selectedClips = clips.filter((c) => c.is_selected);
  const uploadedClips = selectedClips.filter((c) => c.youtube_id || c.upload_status === 'uploaded');
  const queuedClips = selectedClips.filter((c) => c.upload_status === 'queued');
  const errorClips = selectedClips.filter((c) => c.upload_status === 'error');

  return (
    <div className="space-y-5 max-w-lg">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold text-foreground">Concluído</h2>
        {mode === 'queue' ? (
          <p className="text-sm text-subtle">
            ✅ {queuedClips.length > 0 ? queuedClips.length : selectedClips.length} clip
            {selectedClips.length !== 1 ? 's' : ''} salvos na fila.
            Acesse a página de Queue para fazer o upload.
          </p>
        ) : (
          <p className="text-sm text-subtle">
            ✅ {uploadedClips.length}/{selectedClips.length} clip
            {selectedClips.length !== 1 ? 's' : ''} enviados com sucesso.
          </p>
        )}
      </div>

      {/* Direct mode — clip links */}
      {mode === 'direct' && uploadedClips.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-subtle uppercase tracking-wider">Vídeos publicados</p>
          <div className="space-y-2">
            {uploadedClips.map((clip) => (
              <div key={clip.id} className="rounded-lg border border-border bg-card px-4 py-3 flex items-center justify-between gap-3">
                <p className="text-xs text-prose truncate flex-1">{clip.title}</p>
                {clip.youtube_id && (
                  <a
                    href={`https://youtube.com/watch?v=${clip.youtube_id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex-shrink-0 text-xs text-brand hover:text-brand/80 underline underline-offset-2"
                  >
                    Abrir no YouTube ↗
                  </a>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Errors */}
      {errorClips.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-destructive uppercase tracking-wider">Falhas</p>
          <div className="space-y-2">
            {errorClips.map((clip) => (
              <div key={clip.id} className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3">
                <p className="text-xs text-destructive truncate">{clip.title}</p>
                <p className="text-xs text-destructive/80 mt-0.5">Falha no upload — tente novamente pela fila</p>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="flex justify-end pt-2">
        <Button
          type="button"
          variant="secondary"
          onClick={onNewPipeline}
        >
          Novo Pipeline
        </Button>
      </div>
    </div>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export function StepUpload() {
  const goUrl = useAppStore((s) => s.goUrl);
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const run = usePipelineStore((s) => s.run);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const submitUploadConfirm = usePipelineStore((s) => s.submitUploadConfirm);
  const cancelRun = usePipelineStore((s) => s.cancelRun);
  const clearRun = usePipelineStore((s) => s.clearRun);
  const clipProgress = usePipelineStore((s) => s.clipProgress);
  const channels = useChannelStore((s) => s.channels);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [chosenMode, setChosenMode] = useState<'queue' | 'direct' | null>(null);

  const uploadOptions = useMemo<ModeConfig['upload_options'] | undefined>(() => {
    if (!run?.mode_config_json) return undefined;
    try { return (JSON.parse(run.mode_config_json) as ModeConfig).upload_options; }
    catch { return undefined; }
  }, [run?.mode_config_json]);

  const channel = useMemo(
    () => channels.find((c) => c.ID === run?.channel_id) ?? null,
    [channels, run?.channel_id],
  );

  useEffect(() => {
    if (activeRunId == null) return;
    void loadClips(goUrl, activeRunId);
  }, [goUrl, activeRunId]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (channels.length === 0) void fetchChannels(goUrl);
  }, [goUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleConfirm = async (mode: 'queue' | 'direct') => {
    if (!activeRunId) return;
    setSubmitting(true);
    setError(null);
    setChosenMode(mode);
    try {
      await submitUploadConfirm(goUrl, activeRunId, {
        privacy: uploadOptions?.privacy ?? 'private',
        mode,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao confirmar upload');
      setChosenMode(null);
    } finally {
      setSubmitting(false);
    }
  };

  const handleSkip = async () => {
    await handleConfirm('queue');
  };

  const handleCancel = async () => {
    if (!activeRunId) return;
    await cancelRun(goUrl, activeRunId);
  };

  const handleNewPipeline = () => {
    clearRun();
  };

  const state = run?.state ?? 'WAITING_UPLOAD_CONFIRM';

  if (state === 'UPLOADING') {
    return (
      <ProgressScreen
        clips={clips}
        clipProgress={clipProgress}
        onCancel={() => void handleCancel()}
      />
    );
  }

  if (state === 'DONE') {
    return (
      <DoneScreen
        clips={clips}
        mode={chosenMode ?? (uploadOptions?.mode ?? 'queue')}
        onNewPipeline={handleNewPipeline}
      />
    );
  }

  return (
    <ConfirmScreen
      clips={clips}
      uploadOptions={uploadOptions}
      channelName={channel?.ChannelTitle || channel?.Name}
      channelAvatarUrl={channel?.AvatarURL ? `${goUrl}/api/channels/${channel.ID}/avatar` : undefined}
      onConfirm={(mode) => void handleConfirm(mode)}
      onSkip={() => void handleSkip()}
      submitting={submitting}
      error={error}
    />
  );
}
