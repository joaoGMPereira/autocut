'use client';

import { Anton } from 'next/font/google';
import { Sparkles } from 'lucide-react';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

const UPCOMING_FEATURES = [
  'Color grading & LUT application',
  'Audio normalization (LUFS targeting)',
  'Noise reduction & enhancement',
  'Video stabilization',
];

export default function PostOptPage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        POST-OPTIMIZATION
      </h1>

      <div className="rounded-xl bg-card border border-border p-8 flex flex-col items-center gap-6 text-center">
        {/* Icon */}
        <div className="h-14 w-14 rounded-xl bg-[#00D4FF]/10 flex items-center justify-center">
          <Sparkles className="h-7 w-7 text-[#00D4FF]" />
        </div>

        {/* Heading + description */}
        <div className="flex flex-col gap-2">
          <h2 className="text-lg font-semibold">Post-Optimization</h2>
          <p className="text-sm text-[#5C5C80] max-w-md">
            Color grading, audio normalization, and additional post-processing features are coming
            soon.
          </p>
        </div>

        {/* Upcoming features list */}
        <ul className="flex flex-col gap-2 text-left w-full max-w-xs">
          {UPCOMING_FEATURES.map((feature) => (
            <li key={feature} className="flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-[#00D4FF]/40 shrink-0" />
              <span className="text-sm text-[#5C5C80]">{feature}</span>
            </li>
          ))}
        </ul>

        {/* Coming soon badge */}
        <span className="bg-[#00D4FF]/10 text-[#00D4FF] text-xs px-3 py-1 rounded-full font-medium">
          Coming Soon
        </span>
      </div>
    </div>
  );
}
