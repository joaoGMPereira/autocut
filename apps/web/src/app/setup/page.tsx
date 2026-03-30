'use client';

import { useRouter } from 'next/navigation';
import { useSetupCheck } from '@/hooks/useSetupCheck';
import { SetupGate } from '@/components/setup/SetupGate';
import { Loader2 } from 'lucide-react';

export default function SetupPage() {
  const router = useRouter();
  const { isLoading, tools } = useSetupCheck();

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return <SetupGate tools={tools} onComplete={() => router.push('/')} />;
}
