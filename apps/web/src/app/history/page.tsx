import { Anton } from 'next/font/google';
import { RunFilters } from '@/components/history/RunFilters';
import { RunsTable } from '@/components/history/RunsTable';
import { RunDetailPanel } from '@/components/history/RunDetailPanel';
import { HistoryShell } from '@/components/history/HistoryShell';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

export default function HistoryPage() {
  return (
    <main className="flex flex-col gap-6 px-10 py-8 h-full overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1
          className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}
        >
          PIPELINE HISTORY
        </h1>
      </div>

      {/* Client shell: fetches runs on mount and renders filters + table + detail panel */}
      <HistoryShell>
        <RunFilters />
        <RunsTable />
        <RunDetailPanel />
      </HistoryShell>
    </main>
  );
}
