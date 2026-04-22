import DashboardClient from '@/components/dashboard/DashboardClient';

export default function DashboardPage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className="font-display text-[32px] font-bold tracking-[-0.02em] text-heading">
        AutoCut
      </h1>
      <DashboardClient />
    </div>
  );
}
