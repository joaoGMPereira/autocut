'use client';

import { useEffect, useState } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { ClipCard } from '../ClipCard';
import { SubStepProgress } from '../SubStepProgress';
import { GateActionBar } from '../GateActionBar';
import type { UploadConfirmRequest } from '@/types/pipeline';
import { createLogger } from '@/lib/logger';

const log = createLogger('StepUpload');
const goUrl = process.env.NEXT_PUBLIC_GO_URL ?? 'http://localhost:4070';

export function StepUpload() {
  const run = usePipelineStore((s) => s.run);
  const clips = usePipelineStore((s) => s.clips);
  const loadClips = usePipelineStore((s) => s.loadClips);
  const phaseProgress = usePipelineStore((s) => s.phaseProgress);
  const submitUploadConfirm = usePipelineStore((s) => s.submitUploadConfirm);

  const [privacy, setPrivacy] = useState<UploadConfirmRequest['privacy']>('private');

  useEffect(() => {
    if (run?.id) loadClips(goUrl, run.id);
  }, [run?.id, run?.state, loadClips]);

  const isWaitingConfirm = run?.state === 'WAITING_UPLOAD_CONFIRM';
  const isUploading = run?.state === 'UPLOADING';
  const isDone = run?.state === 'DONE';

  async function handleUpload() {
    if (!run || !isWaitingConfirm) return;
    try {
      await submitUploadConfirm(goUrl, run.id, { privacy });
    } catch (err) {
      log.error('[StepUpload] submit failed', { error: err instanceof Error ? err.message : 'Unknown' });
    }
  }

  const selectedClips = clips.filter((c) => c.is_selected);

  return (
    <div className="space-y-4 max-w-2xl">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-1">
          {isDone ? 'Upload Complete' : isUploading ? 'Uploading…' : 'Confirm Upload'}
        </h2>
        <p className="text-sm text-zinc-400">
          {isDone
            ? `${selectedClips.length} clip${selectedClips.length !== 1 ? 's' : ''} uploaded.`
            : isUploading
            ? 'Uploading selected clips to YouTube…'
            : `${selectedClips.length} clip${selectedClips.length !== 1 ? 's' : ''} ready to upload.`}
        </p>
      </div>

      {/* Per-clip upload progress during upload */}
      {isUploading && phaseProgress?.phase === 'upload' && (
        <SubStepProgress
          phase="upload"
          label="Uploading clips"
          icon="⬆"
          percentDone={phaseProgress.percentDone}
          isActive
        />
      )}

      {/* Clips grid */}
      {selectedClips.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {selectedClips.map((c) => (
            <ClipCard key={c.id} clip={c} variant="select" />
          ))}
        </div>
      )}

      {/* Upload confirm controls */}
      {isWaitingConfirm && (
        <GateActionBar>
          <select
            value={privacy}
            onChange={(e) => setPrivacy(e.target.value as UploadConfirmRequest['privacy'])}
            className="bg-zinc-800 border border-border rounded px-2 py-1.5 text-sm text-foreground focus:outline-none focus:border-cyan-500"
          >
            <option value="private">Private</option>
            <option value="unlisted">Unlisted</option>
            <option value="public">Public</option>
          </select>
          <button
            type="button"
            onClick={handleUpload}
            className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-sm font-medium text-white transition-colors"
          >
            Upload →
          </button>
        </GateActionBar>
      )}

      {isDone && selectedClips.some((c) => c.youtube_id) && (
        <div className="space-y-2">
          <p className="text-sm text-emerald-400 font-medium">YouTube Links</p>
          {selectedClips.filter((c) => c.youtube_id).map((c) => (
            <a
              key={c.id}
              href={`https://www.youtube.com/watch?v=${c.youtube_id}`}
              target="_blank"
              rel="noopener noreferrer"
              className="block text-xs text-cyan-400 hover:underline truncate"
            >
              {c.title || `Clip ${c.id}`} → youtube.com/watch?v={c.youtube_id}
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
