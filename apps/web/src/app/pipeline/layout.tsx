import type { ReactNode } from 'react';

export const metadata = {
  title: 'Pipeline — AutoCut',
};

export default function PipelineLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col h-full">
      {children}
    </div>
  );
}
