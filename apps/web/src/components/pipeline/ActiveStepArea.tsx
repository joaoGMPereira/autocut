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
import { StepGeneratingClips } from './steps/StepGeneratingClips';

export function ActiveStepArea() {
  const displayedState = usePipelineStore((s) => s.displayedState());
  const isNavigatedBack = usePipelineStore((s) => s.isNavigatedBack());
  const gateHistory = usePipelineStore((s) => s.gateHistory);
  const run = usePipelineStore((s) => s.run);
  const mode = run?.mode ?? 'ai';

  const historical = isNavigatedBack ? gateHistory[displayedState] : undefined;

  switch (displayedState) {
    case 'WAITING_URL':          return <StepUrl historical={historical} />;
    case 'WAITING_MODE':         return <StepMode historical={historical} />;
    case 'EXECUTING':            return <StepExecute />;
    case 'GENERATING_CLIPS':     return <StepGeneratingClips />;
    case 'WAITING_REVIEW_HIGHLIGHTS': return <StepReviewHighlights historical={historical} />;
    case 'WAITING_THUMBNAIL_CONFIG':  return <StepThumbnailConfig historical={historical} />;
    case 'WAITING_REVIEW_METADATA':   return <StepReviewMetadata historical={historical} />;
    case 'WAITING_REVIEW_CLIPS':      return <StepReviewClips historical={historical} />;
    case 'WAITING_UPLOAD_CONFIRM':
    case 'UPLOADING':
    case 'DONE':                 return <StepUpload />;
    default:                     return null;
  }
}
