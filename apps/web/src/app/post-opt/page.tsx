import { Anton } from 'next/font/google';
import { PostOptTabs } from '@/components/post-opt/PostOptTabs';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

export default function PostOptPage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        POST-OPTIMIZATION
      </h1>
      <PostOptTabs />
    </div>
  );
}
