'use client';

import type { RunState, StepStatus, StepRailEntry } from '@/types/pipeline';
import { StepRailItem } from './StepRailItem';

const STEPS: Array<{ label: string; state: RunState }> = [
  { label: 'URL', state: 'WAITING_URL' },
  { label: 'Mode', state: 'WAITING_MODE' },
  { label: 'Processing', state: 'EXECUTING' },
  { label: 'Review Highlights', state: 'WAITING_REVIEW_HIGHLIGHTS' },
  { label: 'Generating Clips', state: 'GENERATING_CLIPS' },
  { label: 'Thumbnail Config', state: 'WAITING_THUMBNAIL_CONFIG' },
  { label: 'Review Metadata', state: 'WAITING_REVIEW_METADATA' },
  { label: 'Review Clips', state: 'WAITING_REVIEW_CLIPS' },
  { label: 'Upload Confirm', state: 'WAITING_UPLOAD_CONFIRM' },
  { label: 'Uploading', state: 'UPLOADING' },
  { label: 'Done', state: 'DONE' },
];

const STATE_ORDER: RunState[] = STEPS.map((s) => s.state);

function resolveStatus(stepState: RunState, currentState: RunState): StepStatus {
  if (currentState === 'ERROR') {
    const stepIdx = STATE_ORDER.indexOf(stepState);
    const curIdx = STATE_ORDER.indexOf(currentState);
    if (stepIdx < curIdx) return 'done';
    if (stepIdx === curIdx) return 'error';
    return 'idle';
  }
  if (currentState === 'CANCELLED') return 'idle';

  const stepIdx = STATE_ORDER.indexOf(stepState);
  const curIdx = STATE_ORDER.indexOf(currentState);

  if (curIdx < 0) return 'idle';
  if (stepIdx < curIdx) return 'done';
  if (stepIdx === curIdx) {
    // Gate states show as "gate", compute states show as "running"
    const gateStates: RunState[] = [
      'WAITING_URL', 'WAITING_MODE', 'WAITING_REVIEW_HIGHLIGHTS',
      'WAITING_THUMBNAIL_CONFIG', 'WAITING_REVIEW_METADATA',
      'WAITING_REVIEW_CLIPS', 'WAITING_UPLOAD_CONFIRM',
    ];
    return gateStates.includes(currentState) ? 'gate' : 'running';
  }
  return 'idle';
}

interface StepRailProps {
  state: RunState;
}

export function StepRail({ state }: StepRailProps) {
  const entries: StepRailEntry[] = STEPS.map((s) => ({
    label: s.label,
    state: s.state,
    status: resolveStatus(s.state, state),
  }));

  return (
    <div className="py-4">
      <p className="px-4 pb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">
        Pipeline
      </p>
      {entries.map((entry, i) => (
        <StepRailItem
          key={entry.state}
          label={entry.label}
          status={entry.status}
          isLast={i === entries.length - 1}
        />
      ))}
    </div>
  );
}
