'use client';

import { Anton } from 'next/font/google';
import { Brain } from 'lucide-react';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

const upcomingFeatures = [
  'Batch title templates with dynamic variables',
  'Description variables and reusable snippets',
  'Tag management across multiple videos',
  'Scheduled bulk updates',
] as const;

export default function MassUpdatePage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        MASS UPDATE
      </h1>

      <div className="rounded-xl bg-card border border-border p-8 flex flex-col items-center gap-6 text-center">
        <div className="h-14 w-14 rounded-xl bg-[#00D4FF]/10 flex items-center justify-center">
          <Brain className="h-7 w-7 text-[#00D4FF]" />
        </div>

        <div className="flex flex-col gap-2">
          <h2 className="text-lg font-semibold">Mass Update</h2>
          <p className="text-sm text-[#5C5C80] max-w-md">
            Bulk update titles, descriptions, and tags for multiple videos at once.
          </p>
        </div>

        <ul className="flex flex-col gap-2 text-left">
          {upcomingFeatures.map((feature) => (
            <li key={feature} className="flex items-center gap-2 text-sm text-[#5C5C80]">
              <span className="h-1 w-1 rounded-full bg-[#5C5C80] shrink-0" />
              {feature}
            </li>
          ))}
        </ul>

        <span className="bg-[#00D4FF]/10 text-[#00D4FF] text-xs px-3 py-1 rounded-full font-medium">
          Coming Soon
        </span>
      </div>
    </div>
  );
}
