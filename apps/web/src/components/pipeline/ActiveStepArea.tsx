'use client';

import type { ReactNode } from 'react';
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

  let stepElement: ReactNode = null;
  switch (displayedState) {
    case 'WAITING_URL':          stepElement = <StepUrl historical={historical} />; break;
    case 'WAITING_MODE':         stepElement = <StepMode historical={historical} />; break;
    case 'EXECUTING':            stepElement = <StepExecute />; break;
    case 'GENERATING_CLIPS':     stepElement = <StepGeneratingClips />; break;
    case 'WAITING_REVIEW_HIGHLIGHTS': stepElement = <StepReviewHighlights historical={historical} />; break;
    case 'WAITING_THUMBNAIL_CONFIG':  stepElement = <StepThumbnailConfig historical={historical} />; break;
    case 'WAITING_REVIEW_METADATA':   stepElement = <StepReviewMetadata historical={historical} />; break;
    case 'WAITING_REVIEW_CLIPS':      stepElement = <StepReviewClips historical={historical} />; break;
    case 'WAITING_UPLOAD_CONFIRM':
    case 'UPLOADING':
    case 'DONE':                 stepElement = <StepUpload />; break;
    default:                     stepElement = null;
  }
  return (
    <div data-testid="active-step" data-step-state={displayedState}>
      {stepElement}
    </div>
  );
}
