import { Anton } from 'next/font/google';
import Link from 'next/link';
import {
  Download,
  Users,
  Wrench,
  ListOrdered,
  Play,
  Clapperboard,
  Zap,
  Image,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

const anton = Anton({
  weight: '400',
  subsets: ['latin'],
  display: 'swap',
});

const GO_URL = process.env.NEXT_PUBLIC_GO_URL ?? 'http://127.0.0.1:4070';

interface GoToolStatus {
  name: string;
  installed: boolean;
  required: boolean;
}

interface GoChannel {
  ID: number;
  Name: string;
  ChannelID: string;
  ChannelTitle: string;
}

async function getSetupStats(): Promise<{ installed: number; total: number }> {
  try {
    const res = await fetch(`${GO_URL}/api/setup/status`, { cache: 'no-store' });
    if (!res.ok) return { installed: 0, total: 0 };
    const data = (await res.json()) as { tools: GoToolStatus[] };
    const tools = data.tools ?? [];
    return { installed: tools.filter((t) => t.installed).length, total: tools.length };
  } catch {
    return { installed: 0, total: 0 };
  }
}

async function getChannels(): Promise<GoChannel[]> {
  try {
    const res = await fetch(`${GO_URL}/api/channels`, { cache: 'no-store' });
    if (!res.ok) return [];
    return (await res.json()) as GoChannel[];
  } catch {
    return [];
  }
}

async function getUploadStats(): Promise<{ active: number; pending: number }> {
  try {
    const res = await fetch(`${GO_URL}/api/upload`, { cache: 'no-store' });
    if (!res.ok) return { active: 0, pending: 0 };
    const items = (await res.json()) as Array<{ status?: string }>;
    return {
      active: items.filter((i) => i.status === 'active').length,
      pending: items.filter((i) => !i.status || i.status === 'pending').length,
    };
  } catch {
    return { active: 0, pending: 0 };
  }
}

const quickActions: Array<{ label: string; href: string; icon: LucideIcon }> = [
  { label: 'New Download', href: '/pipeline', icon: Play },
  { label: 'Process Shorts', href: '/shorts', icon: Clapperboard },
  { label: 'Optimize Video', href: '/optimizer', icon: Zap },
  { label: 'Generate Thumbnail', href: '/thumbnail', icon: Image },
];

export default async function DashboardPage() {
  const [setup, channels, uploads] = await Promise.all([
    getSetupStats(),
    getChannels(),
    getUploadStats(),
  ]);

  const stats: Array<{ label: string; value: string | number; icon: LucideIcon }> = [
    { label: 'Downloads ativos', value: uploads.active, icon: Download },
    { label: 'Canais', value: channels.length, icon: Users },
    { label: 'Ferramentas', value: `${setup.installed}/${setup.total}`, icon: Wrench },
    { label: 'Queue', value: uploads.pending, icon: ListOrdered },
  ];

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      {/* Header */}
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        DASHBOARD
      </h1>

      {/* Status cards */}
      <div className="grid grid-cols-4 gap-4">
        {stats.map(({ label, value, icon: Icon }) => (
          <div
            key={label}
            className="rounded-xl bg-card border border-border p-5 flex items-center gap-4"
          >
            <Icon className="h-5 w-5 text-[#00D4FF] shrink-0" />
            <div className="min-w-0">
              <p className="text-xs text-[#5C5C80] leading-none mb-1">{label}</p>
              <p className="font-mono text-2xl font-semibold tabular-nums leading-none">
                {value}
              </p>
            </div>
          </div>
        ))}
      </div>

      {/* Quick Actions */}
      <div className="flex flex-col gap-3">
        <h2 className="text-[18px] font-semibold">Quick Actions</h2>
        <div className="grid grid-cols-2 gap-3">
          {quickActions.map(({ label, href, icon: Icon }) => (
            <Link
              key={href}
              href={href}
              className="h-14 rounded-xl bg-card border border-border hover:border-[#00D4FF]/40 hover:bg-[#1A1A26] transition-colors flex items-center gap-3 px-4"
            >
              <Icon className="h-4 w-4 text-[#00D4FF] shrink-0" />
              <span className="text-sm font-medium">{label}</span>
            </Link>
          ))}
        </div>
      </div>

      {/* Channels — only shown when at least one is configured */}
      {channels.length > 0 && (
        <div className="flex flex-col gap-3">
          <h2 className="text-[18px] font-semibold">Channels</h2>
          <div className="rounded-xl bg-card border border-border divide-y divide-border">
            {channels.map((ch) => (
              <div key={ch.ID} className="flex items-center justify-between px-5 py-3">
                <span className="text-sm font-medium">{ch.Name || ch.ChannelTitle}</span>
                <span className="text-xs text-[#5C5C80] bg-[#1A1A26] px-2 py-0.5 rounded-md">
                  YouTube
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
