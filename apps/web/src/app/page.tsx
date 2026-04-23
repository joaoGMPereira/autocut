import DashboardClient from '@/components/dashboard/DashboardClient';
import { PageHeader } from '@/components/ui/page-header';

export default function DashboardPage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <PageHeader title="AutoCut" />
      <DashboardClient />
    </div>
  );
}
