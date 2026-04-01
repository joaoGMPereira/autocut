'use client';

import { useWizardStore } from '@/store/wizardStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { PipelineWizardLayout } from '@/components/pipeline/PipelineWizardLayout';
import { UrlStepPanel } from '@/components/pipeline/steps/UrlStepPanel';
import { ModeStepPanel } from '@/components/pipeline/steps/ModeStepPanel';
import { ExecuteStepPanel } from '@/components/pipeline/steps/ExecuteStepPanel';
import { ShortsStep } from '@/components/pipeline/steps/ShortsStep';
import { ThumbnailStep } from '@/components/pipeline/steps/ThumbnailStep';
import { UploadStep } from '@/components/pipeline/steps/UploadStep';
import type { StepStatus } from '@/components/pipeline/StepCard';

function useStepStatus(stepName: string): StepStatus {
  const run = usePipelineStore((s) => s.run);
  if (!run) return 'idle';
  const step = run.steps.find((s) => s.step === stepName);
  if (!step) return 'idle';
  if (step.status === 'running') return 'running';
  if (step.status === 'done' || step.status === 'completed') return 'done';
  if (step.status === 'error' || step.status === 'failed') return 'error';
  if (step.status === 'skipped') return 'skipped';
  return 'idle';
}

export default function PipelinePage() {
  const stepDef = useWizardStore((s) => s.steps[s.currentStep]);

  const shortsStatus = useStepStatus('shorts');
  const thumbnailStatus = useStepStatus('thumbnail');
  const uploadStatus = useStepStatus('upload');

  function renderStep() {
    switch (stepDef?.id) {
      case 'url':
        return <UrlStepPanel />;
      case 'mode':
        return <ModeStepPanel />;
      case 'execute':
        return <ExecuteStepPanel />;
      case 'highlights':
        return <div className="text-[#A0A0B8] text-sm p-4">Highlights review — em breve</div>;
      case 'metadata':
        return <div className="text-[#A0A0B8] text-sm p-4">Edicao de metadados — em breve</div>;
      case 'clips':
        return <ShortsStep status={shortsStatus} />;
      case 'upload':
        return <UploadStep status={uploadStatus} />;
      default:
        return <div className="text-[#5C5C80] text-sm">Step nao encontrado</div>;
    }
  }

  return (
    <PipelineWizardLayout>
      {renderStep()}
    </PipelineWizardLayout>
  );
}
