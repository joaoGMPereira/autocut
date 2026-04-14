import { DownloadInfoCard } from '../DownloadInfoCard';

export function StepExecute() {
  return (
    <div className="space-y-2 max-w-lg">
      <h2 className="text-xl font-semibold text-foreground">Processing</h2>
      <DownloadInfoCard />
    </div>
  );
}
