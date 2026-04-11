import { Anton } from 'next/font/google';
import DashboardClient from '@/components/dashboard/DashboardClient';

const anton = Anton({
  weight: '400',
  subsets: ['latin'],
  display: 'swap',
});

export default function DashboardPage() {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        AUTOCUT DASHBOARD
      </h1>
      <DashboardClient />
    </div>
  );
}
