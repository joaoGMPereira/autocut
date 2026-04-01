'use client';

import type { ReactNode } from 'react';
import { useCallback, useEffect } from 'react';
import { Anton } from 'next/font/google';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { useWizardStore } from '@/store/wizardStore';
import { usePipelineStore } from '@/store/pipelineStore';
import type { PipelineRun } from '@/store/pipelineStore';
import { useAppStore } from '@/store/appStore';
import { useDispatcherStore } from '@/store/dispatcherStore';
import { startBackgroundDownload } from '@/lib/backgroundDownload';
import { PipelineStepIndicator } from './PipelineStepIndicator';
import { DownloadProgressBar } from './DownloadProgressBar';
import { RecentRunsSection } from './RecentRunsSection';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

interface PipelineWizardLayoutProps {
  children: ReactNode;
}

export function PipelineWizardLayout({ children }: PipelineWizardLayoutProps) {
  const {
    steps,
    currentStep,
    maxReached,
    canAdvance,
    visibleSteps,
    goNext,
    goBack,
    goToStep,
    jumpToStep,
    resetWizard,
    pendingUrl,
    pendingModeConfig,
  } = useWizardStore();

  const { activeRunId, run, runs, clearRun, createRun, patchRunMode, loadRun, listRuns, deleteRun } = usePipelineStore();
  const goUrl = useAppStore((s) => s.goUrl);
  const resetDispatcher = useDispatcherStore((s) => s.reset);

  // Load run history on mount (once goUrl is available)
  useEffect(() => {
    if (goUrl && !activeRunId) {
      void listRuns(goUrl);
    }
  }, [goUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  const visible = visibleSteps();
  const currentIdx = visible.findIndex(
    (s) => s.id === steps[currentStep]?.id,
  );
  const isFirstStep = currentIdx === 0;
  const isLastStep = currentIdx === visible.length - 1;

  const handleNewPipeline = () => {
    resetWizard();
    clearRun();
    resetDispatcher();
  };

  // Resume a previous run: load from API, hydrate dispatcher, navigate to appropriate step.
  const handleResumeRun = useCallback(
    async (runId: number) => {
      resetWizard();
      resetDispatcher();
      await loadRun(goUrl, runId);
      // After loadRun, pipelineStore has the full run (with stepOutputs + hydrated dispatcher).
      const loadedRun = usePipelineStore.getState().run as PipelineRun | null;
      if (!loadedRun) return;

      // Determine which step to jump to
      let targetIndex = 1; // default: mode step
      const hasMode = loadedRun.mode && loadedRun.mode !== '';
      if (hasMode && ['in_progress', 'completed', 'failed'].includes(loadedRun.status)) {
        targetIndex = 2; // execute step
      }
      jumpToStep(targetIndex);
    },
    [goUrl, loadRun, resetWizard, resetDispatcher, jumpToStep],
  );

  // Ao sair do URL step: criar run imediatamente + iniciar download em background
  // Ao sair do Modo: PATCH run com mode config
  const handleNext = useCallback(async () => {
    const currentStepId = steps[currentStep]?.id;

    if (currentStepId === 'url' && !activeRunId && pendingUrl) {
      try {
        const runId = await createRun(goUrl, pendingUrl, '');
        await startBackgroundDownload(goUrl, runId, pendingUrl);
      } catch {
        // errors visible in store; don't block navigation
      }
    }

    if (currentStepId === 'mode' && activeRunId && pendingModeConfig) {
      try {
        await patchRunMode(goUrl, activeRunId, pendingModeConfig);
      } catch {
        // non-blocking
      }
    }

    goNext();
  }, [steps, currentStep, activeRunId, pendingUrl, pendingModeConfig, createRun, patchRunMode, goUrl, goNext, resetDispatcher]);

  return (
    <div className="flex flex-col gap-6 px-10 py-8 h-full">
      {/* Header row */}
      <div className="flex items-center justify-between">
        <h1
          className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}
        >
          PIPELINE
        </h1>
        <Button variant="secondary" size="sm" onClick={handleNewPipeline}>
          Novo Pipeline
        </Button>
      </div>

      {/* Run info bar */}
      {activeRunId != null && run && (
        <div className="flex items-center gap-4 rounded-lg bg-[#12121C] border border-border px-4 py-2.5">
          <span className="bg-[#1A1A26] px-2 py-1 rounded-md text-xs text-[#00D4FF] font-medium">
            Run #{run.id}
          </span>
          <span className="text-xs text-[#A0A0B8]">{run.status}</span>
          {run.progress > 0 && (
            <div className="flex items-center gap-2 flex-1 max-w-xs">
              <Progress value={run.progress} className="h-1.5" />
              <span className="text-xs text-[#5C5C80]">{run.progress}%</span>
            </div>
          )}
        </div>
      )}

      {/* Background download status bar */}
      <DownloadProgressBar />

      {/* Step indicator */}
      <PipelineStepIndicator
        steps={steps}
        currentStep={currentStep}
        maxReached={maxReached}
        onStepClick={goToStep}
      />

      {/* Recent runs — shown when no active run */}
      {activeRunId == null && runs.length > 0 && (
        <RecentRunsSection
          runs={runs}
          onResume={handleResumeRun}
          onDelete={(id) => void deleteRun(goUrl, id)}
        />
      )}

      {/* Content area */}
      <div className="flex-1 min-h-0 overflow-y-auto">{children}</div>

      {/* Footer navigation */}
      {!isLastStep && (
        <div className="flex items-center pt-2 border-t border-border">
          <Button
            variant="secondary"
            onClick={goBack}
            disabled={isFirstStep}
          >
            Voltar
          </Button>
          <div className="flex-1" />
          <Button
            className="bg-[#00D4FF] text-[#0D0D15] hover:bg-[#00D4FF]/90 font-semibold"
            onClick={handleNext}
            disabled={!canAdvance}
          >
            Proximo
          </Button>
        </div>
      )}
    </div>
  );
}
