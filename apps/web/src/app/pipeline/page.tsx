import { PipelineShell } from '@/components/pipeline/PipelineShell';
import { ActiveStepArea } from '@/components/pipeline/ActiveStepArea';

export default function PipelinePage() {
  return (
    <PipelineShell>
      <ActiveStepArea />
    </PipelineShell>
  );
}
