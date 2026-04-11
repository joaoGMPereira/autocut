'use client';

import { usePipelineStore } from '@/store/pipelineStore';
import { StepUrl } from './steps/StepUrl';
import { StepMode } from './steps/StepMode';
import { StepExecute } from './steps/StepExecute';
import { StepReviewHighlights } from './steps/StepReviewHighlights';
import { StepThumbnailConfig } from './steps/StepThumbnailConfig';
import { StepReviewMetadata } from './steps/StepReviewMetadata';
import { StepReviewClips } from './steps/StepReviewClips';
import { StepUpload } from './steps/StepUpload';

export function ActiveStepArea() {
  const run = usePipelineStore((s) => s.run);
  const state = run?.state ?? 'WAITING_URL';

  switch (state) {
    case 'WAITING_URL':
      return <StepUrl />;
    case 'WAITING_MODE':
      return <StepMode />;
    case 'EXECUTING':
    case 'GENERATING_CLIPS':
      return <StepExecute />;
    case 'WAITING_REVIEW_HIGHLIGHTS':
      return <StepReviewHighlights />;
    case 'WAITING_THUMBNAIL_CONFIG':
      return <StepThumbnailConfig />;
    case 'WAITING_REVIEW_METADATA':
      return <StepReviewMetadata />;
    case 'WAITING_REVIEW_CLIPS':
      return <StepReviewClips />;
    case 'WAITING_UPLOAD_CONFIRM':
    case 'UPLOADING':
    case 'DONE':
      return <StepUpload />;
    case 'ERROR':
      return (
        <div className="flex flex-col items-center justify-center h-64 gap-3">
          <div className="text-red-500 text-4xl">⚠</div>
          <p className="text-red-400 font-medium">Pipeline Error</p>
          <p className="text-zinc-400 text-sm text-center max-w-sm">{run?.error || 'An unexpected error occurred.'}</p>
        </div>
      );
    case 'CANCELLED':
      return (
        <div className="flex flex-col items-center justify-center h-64 gap-3">
          <div className="text-zinc-500 text-4xl">✕</div>
          <p className="text-zinc-400 font-medium">Run Cancelled</p>
        </div>
      );
    default:
      return null;
  }
}
