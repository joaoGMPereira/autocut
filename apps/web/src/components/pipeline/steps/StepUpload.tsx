'use client';

import { useEffect, useMemo, useState } from 'react';
import { useAppStore } from '@/store/appStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { useChannelStore } from '@/store/channelStore';
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
  const map: Record<string, { label: string; cls: string }> = {
    private:  { label: '🔒 Private',  cls: 'bg-zinc-800 text-zinc-400 border-zinc-700' },
    unlisted: { label: '🔗 Unlisted', cls: 'bg-amber-950 text-amber-400 border-amber-800' },
    public:   { label: '🌐 Public',   cls: 'bg-emerald-950 text-emerald-400 border-emerald-800' },
  };
  const { label, cls } = map[privacy] ?? map['private'];
  return (
    <span className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium ${cls}`}>
      {label}
    </span>
  );
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
        <p className="text-sm text-zinc-500">Revise as configurações antes de prosseguir.</p>
      </div>

      {/* Summary card */}
      <div className="rounded-lg border border-zinc-700 bg-zinc-900 divide-y divide-zinc-800">
        {/* Channel */}
        {channelName && (
          <div className="flex items-center gap-3 px-4 py-3">
            {channelAvatarUrl && (
              <img src={channelAvatarUrl} alt="" className="h-8 w-8 rounded-full object-cover flex-shrink-0" />
            )}
            <div className="min-w-0">
              <p className="text-xs text-zinc-500">Canal</p>
              <p className="text-sm font-medium text-zinc-200 truncate">{channelName}</p>
            </div>
          </div>
        )}

        {/* Clip count & duration */}
        <div className="grid grid-cols-2 divide-x divide-zinc-800">
          <div className="px-4 py-3">
            <p className="text-xs text-zinc-500">Clips selecionados</p>
            <p className="text-lg font-semibold text-zinc-100">{selectedClips.length}</p>
          </div>
          <div className="px-4 py-3">
            <p className="text-xs text-zinc-500">Duração total</p>
            <p className="text-lg font-semibold text-zinc-100">{fmtDuration(totalDurationSec)}</p>
          </div>
        </div>

        {/* Privacy */}
        <div className="flex items-center justify-between px-4 py-3">
          <p className="text-xs text-zinc-500">Privacidade</p>
          <PrivacyBadge privacy={privacy} />
        </div>
      </div>

      {/* Mode selector */}
      <div className="space-y-2">
        <p className="text-xs font-medium text-zinc-400 uppercase tracking-wider">Modo de Upload</p>
        <div className="grid grid-cols-2 gap-3">
          <button
            onClick={() => setMode('queue')}
            className={[
              'flex flex-col gap-1.5 rounded-xl border p-4 text-left transition-all',
              mode === 'queue'
                ? 'border-zinc-300 bg-zinc-800 ring-1 ring-zinc-300'
                : 'border-zinc-700 bg-zinc-900 hover:border-zinc-600 hover:bg-zinc-800/50',
            ].join(' ')}
          >
            {mode === 'queue' && (
              <span className="self-end flex h-4 w-4 items-center justify-center rounded-full bg-zinc-300 text-zinc-900 text-[10px] font-bold">✓</span>
            )}
            <span className="text-base">🗄️</span>
            <span className="text-sm font-semibold text-zinc-200">Salvar na Fila</span>
            <span className="text-xs text-zinc-500 leading-relaxed">
              Salva os clips para upload posterior pela página de Queue.
            </span>
          </button>

          <button
            onClick={() => setMode('direct')}
            className={[
              'flex flex-col gap-1.5 rounded-xl border p-4 text-left transition-all',
              mode === 'direct'
                ? 'border-zinc-300 bg-zinc-800 ring-1 ring-zinc-300'
                : 'border-zinc-700 bg-zinc-900 hover:border-zinc-600 hover:bg-zinc-800/50',
            ].join(' ')}
          >
            {mode === 'direct' && (
              <span className="self-end flex h-4 w-4 items-center justify-center rounded-full bg-zinc-300 text-zinc-900 text-[10px] font-bold">✓</span>
            )}
            <span className="text-base">🚀</span>
            <span className="text-sm font-semibold text-zinc-200">Upload Direto</span>
            <span className="text-xs text-zinc-500 leading-relaxed">
              Faz o upload agora diretamente para o YouTube com progresso em tempo real.
            </span>
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-md border border-red-800 bg-red-950 px-4 py-3 text-xs text-red-400">
          {error}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between pt-1">
        <button
          type="button"
          onClick={onSkip}
          disabled={submitting}
          className="text-xs text-zinc-600 hover:text-zinc-400 underline underline-offset-2 disabled:opacity-50 transition-colors"
        >
          Pular Upload
        </button>

        <button
          type="button"
          onClick={() => onConfirm(mode)}
          disabled={submitting || selectedClips.length === 0}
          className="px-5 py-2 rounded-lg bg-zinc-100 hover:bg-white text-zinc-900 text-sm font-semibold disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {submitting
            ? 'Processando…'
            : mode === 'direct'
              ? `Iniciar Upload (${selectedClips.length} clip${selectedClips.length !== 1 ? 's' : ''})`
              : `Salvar na Fila (${selectedClips.length} clip${selectedClips.length !== 1 ? 's' : ''})`}
        </button>
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
        <p className="text-sm text-zinc-500">
          {uploadedCount}/{selectedClips.length} clips enviados
        </p>
      </div>

      {/* Overall progress bar */}
      <div className="space-y-1.5">
        <div className="flex items-center justify-between text-xs text-zinc-500">
          <span>Progresso geral</span>
          <span>{overallPct}%</span>
        </div>
        <div className="h-2 w-full rounded-full bg-zinc-800 overflow-hidden">
          <div
            className="h-full rounded-full bg-zinc-300 transition-all duration-300"
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
              className="rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-3 space-y-2"
            >
              <div className="flex items-start justify-between gap-2">
                <p className="text-xs font-medium text-zinc-300 truncate flex-1">{clip.title}</p>
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
                  <span className="flex-shrink-0 text-xs text-red-400">Erro</span>
                )}
              </div>

              {!isDone && !isError && (
                <>
                  <div className="h-1.5 w-full rounded-full bg-zinc-800 overflow-hidden">
                    <div
                      className="h-full rounded-full bg-zinc-400 transition-all duration-200"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <div className="flex items-center justify-between text-[11px] text-zinc-600">
                    <span>{pct}%</span>
                  </div>
                </>
              )}

              {isDone && (
                <div className="text-[11px] text-emerald-500 font-medium">✓ Enviado</div>
              )}
            </div>
          );
        })}
      </div>

      <div className="flex justify-end">
        <button
          type="button"
          onClick={onCancel}
          className="text-xs text-zinc-600 hover:text-red-400 underline underline-offset-2 transition-colors"
        >
          Cancelar upload
        </button>
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
          <p className="text-sm text-zinc-400">
            ✅ {queuedClips.length > 0 ? queuedClips.length : selectedClips.length} clip
            {selectedClips.length !== 1 ? 's' : ''} salvos na fila.
            Acesse a página de Queue para fazer o upload.
          </p>
        ) : (
          <p className="text-sm text-zinc-400">
            ✅ {uploadedClips.length}/{selectedClips.length} clip
            {selectedClips.length !== 1 ? 's' : ''} enviados com sucesso.
          </p>
        )}
      </div>

      {/* Direct mode — clip links */}
      {mode === 'direct' && uploadedClips.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-zinc-400 uppercase tracking-wider">Vídeos publicados</p>
          <div className="space-y-2">
            {uploadedClips.map((clip) => (
              <div key={clip.id} className="rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-3 flex items-center justify-between gap-3">
                <p className="text-xs text-zinc-300 truncate flex-1">{clip.title}</p>
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
          <p className="text-xs font-medium text-red-400 uppercase tracking-wider">Falhas</p>
          <div className="space-y-2">
            {errorClips.map((clip) => (
              <div key={clip.id} className="rounded-lg border border-red-900 bg-red-950/50 px-4 py-3">
                <p className="text-xs text-red-300 truncate">{clip.title}</p>
                <p className="text-xs text-red-500 mt-0.5">Falha no upload — tente novamente pela fila</p>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="flex justify-end pt-2">
        <button
          type="button"
          onClick={onNewPipeline}
          className="px-5 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-sm font-medium transition-colors"
        >
          Novo Pipeline
        </button>
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
