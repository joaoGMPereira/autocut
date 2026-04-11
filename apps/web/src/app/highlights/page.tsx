import { Anton } from 'next/font/google';
import { HighlightsClient } from '@/components/highlights/HighlightsClient';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

export default function HighlightsPage() {
  return (
    <main className="flex flex-col gap-6 px-10 py-8 h-full overflow-y-auto">
      <div className="flex items-center justify-between">
        <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
          HIGHLIGHTS
        </h1>
      </div>

      <HighlightsClient />
    </main>
  );
}
