'use client';

import type { WizardStepDef } from '@/store/wizardStore';

interface PipelineStepIndicatorProps {
  steps: WizardStepDef[];
  currentStep: number;
  maxReached: number;
  onStepClick?: (index: number) => void;
}

export function PipelineStepIndicator({
  steps,
  currentStep,
  maxReached,
  onStepClick,
}: PipelineStepIndicatorProps) {
  // Build visible steps with their original (full-array) indices
  const visible = steps
    .map((step, index) => ({ step, index }))
    .filter(({ step }) => !step.hidden);

  return (
    <div className="flex items-start justify-between w-full">
      {visible.map(({ step, index }, visibleIdx) => {
        const isActive = index === currentStep;
        const isDone = index < currentStep && index <= maxReached;
        const isReachable = !isActive && !isDone && index <= maxReached;
        const isLocked = index > maxReached;
        const clickable = index <= maxReached && !isActive;

        return (
          <div key={step.id} className="flex items-start flex-1 last:flex-none">
            {/* Step circle + label */}
            <button
              type="button"
              disabled={!clickable}
              onClick={() => clickable && onStepClick?.(index)}
              className={`flex flex-col items-center gap-1.5 ${
                clickable ? 'cursor-pointer' : 'cursor-default'
              }`}
            >
              <div
                className={`flex items-center justify-center w-8 h-8 rounded-full text-xs font-semibold transition-colors ${
                  isActive
                    ? 'bg-[#00D4FF] text-white'
                    : isDone
                      ? 'bg-emerald-500 text-white'
                      : isReachable
                        ? 'border border-[#00D4FF]/40 text-[#A0A0B8]'
                        : 'bg-[#1A1A26] text-[#5C5C80]'
                }`}
              >
                {isDone ? '\u2713' : visibleIdx + 1}
              </div>
              <span
                className={`text-[11px] font-medium whitespace-nowrap ${
                  isActive
                    ? 'text-[#F0F0F8]'
                    : isLocked
                      ? 'text-[#5C5C80]'
                      : 'text-[#A0A0B8]'
                }`}
              >
                {step.label}
              </span>
            </button>

            {/* Connector line (not after the last visible step) */}
            {visibleIdx < visible.length - 1 && (
              <div className="flex-1 flex items-center pt-4 px-2">
                <div
                  className={`h-px w-full ${
                    isDone
                      ? 'border-t border-[#00D4FF]'
                      : 'border-t border-dashed border-border'
                  }`}
                />
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
