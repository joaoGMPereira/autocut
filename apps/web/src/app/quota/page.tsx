'use client';

import { useEffect } from 'react';
import { Anton } from 'next/font/google';
import { useQuotaStore, type QuotaBreakdown, type QuotaHistory } from '@/store/quotaStore';

const anton = Anton({
  weight: '400',
  subsets: ['latin'],
  display: 'swap',
});

interface QuotaBarProps {
  label: string;
  used: number;
  limit: number;
}

function QuotaBar({ label, used, limit }: QuotaBarProps) {
  const pct = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
  const color =
    pct > 80 ? 'bg-red-500' : pct > 60 ? 'bg-yellow-500' : 'bg-emerald-500';
  return (
    <div>
      <div className="flex justify-between text-sm mb-1 text-[#A0A0C0]">
        <span>{label}</span>
        <span>
          {used}/{limit}
        </span>
      </div>
      <div className="h-2 bg-white/5 rounded-full">
        <div
          className={`h-full rounded-full transition-all ${color}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

interface MiniBarChartProps {
  history: QuotaHistory[];
}

function MiniBarChart({ history }: MiniBarChartProps) {
  const last7 = [...history].slice(0, 7).reverse();
  const max = Math.max(...last7.map((d) => d.units_used), 1);

  return (
    <div className="flex items-end gap-2 h-24">
      {last7.map((entry) => {
        const heightPct = (entry.units_used / max) * 100;
        return (
          <div key={entry.date} className="flex flex-col items-center gap-1 flex-1">
            <div className="w-full flex items-end" style={{ height: '80px' }}>
              <div
                className="w-full bg-[#00D4FF]/40 rounded-t transition-all hover:bg-[#00D4FF]/70"
                style={{ height: `${heightPct}%` }}
                title={`${entry.units_used} units`}
              />
            </div>
            <span className="text-[10px] text-[#5C5C80] leading-none">
              {entry.date.slice(5)}
            </span>
          </div>
        );
      })}
    </div>
  );
}

interface BreakdownSectionProps {
  breakdown: QuotaBreakdown;
}

function BreakdownSection({ breakdown }: BreakdownSectionProps) {
  return (
    <div className="flex flex-col gap-4">
      <QuotaBar
        label="Uploads"
        used={breakdown.uploads_used}
        limit={breakdown.uploads_limit}
      />
      <QuotaBar
        label="Updates"
        used={breakdown.updates_used}
        limit={breakdown.updates_limit}
      />
      <QuotaBar
        label="Reads"
        used={breakdown.reads_used}
        limit={breakdown.reads_limit}
      />
      <QuotaBar
        label="Total Units"
        used={breakdown.total_units}
        limit={breakdown.total_limit}
      />
    </div>
  );
}

export default function QuotaPage() {
  const { breakdown, history, loading, fetchBreakdown, fetchHistory, resetQuota } =
    useQuotaStore();

  useEffect(() => {
    void fetchBreakdown();
    void fetchHistory();
  }, [fetchBreakdown, fetchHistory]);

  const handleReset = () => {
    if (!confirm('Reset all quota counters for today? This cannot be undone.')) {
      return;
    }
    void resetQuota().then(() => {
      void fetchBreakdown();
    });
  };

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1
        className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}
      >
        QUOTA MONITOR
      </h1>

      {loading && !breakdown && (
        <p className="text-[#5C5C80] text-sm">Loading…</p>
      )}

      {breakdown && (
        <section className="bg-[#0D0D1A] border border-white/5 rounded-xl p-6 flex flex-col gap-6">
          <h2 className="text-[#A0A0C0] text-xs font-semibold uppercase tracking-wider">
            Today's Usage
          </h2>
          <BreakdownSection breakdown={breakdown} />
        </section>
      )}

      {history.length > 0 && (
        <section className="bg-[#0D0D1A] border border-white/5 rounded-xl p-6 flex flex-col gap-4">
          <h2 className="text-[#A0A0C0] text-xs font-semibold uppercase tracking-wider">
            Usage last 7 days
          </h2>
          <MiniBarChart history={history} />
        </section>
      )}

      <div>
        <button
          type="button"
          onClick={handleReset}
          className="px-4 py-2 rounded-md bg-red-900/40 text-red-400 border border-red-800/40 text-sm hover:bg-red-900/70 transition-colors"
        >
          Reset Today's Count
        </button>
      </div>
    </div>
  );
}
