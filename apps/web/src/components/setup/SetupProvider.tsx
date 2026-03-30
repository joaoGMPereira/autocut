'use client';

import { useEffect, useState, useCallback } from 'react';
import { Loader2 } from 'lucide-react';
import { useSetupCheck } from '@/hooks/useSetupCheck';
import { SetupGate } from './SetupGate';

const BACKEND_TIMEOUT_MS = 3_000;

interface SetupProviderProps {
  children: React.ReactNode;
}

export function SetupProvider({ children }: SetupProviderProps) {
  const { isReady, isLoading, tools, missingRequired, error } =
    useSetupCheck();
  const [timedOut, setTimedOut] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  // 3-second timeout — don't block the app if the backend is offline
  useEffect(() => {
    if (!isLoading) {
      setTimedOut(false);
      return;
    }

    const timer = setTimeout(() => {
      setTimedOut(true);
    }, BACKEND_TIMEOUT_MS);

    return () => clearTimeout(timer);
  }, [isLoading]);

  const handleComplete = useCallback(() => {
    setDismissed(true);
  }, []);

  // Loading: show spinner until timeout
  if (isLoading && !timedOut) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Backend offline after timeout — let the app load with a warning
  if (timedOut && isLoading) {
    return (
      <div className="relative h-full">
        <div className="absolute top-4 left-1/2 -translate-x-1/2 z-50 rounded-lg border border-amber-400/30 bg-amber-400/10 px-4 py-2 text-xs text-amber-400">
          Backend is starting up. Some features may be unavailable.
        </div>
        {children}
      </div>
    );
  }

  // Error (backend returned an error) — non-blocking warning
  if (error && tools.length === 0) {
    return (
      <div className="relative h-full">
        <div className="absolute top-4 left-1/2 -translate-x-1/2 z-50 rounded-lg border border-amber-400/30 bg-amber-400/10 px-4 py-2 text-xs text-amber-400">
          Could not check tool status: {error}
        </div>
        {children}
      </div>
    );
  }

  // Missing required tools — show SetupGate
  if (missingRequired.length > 0 && !dismissed) {
    return <SetupGate tools={tools} onComplete={handleComplete} />;
  }

  // All good
  return <>{children}</>;
}
