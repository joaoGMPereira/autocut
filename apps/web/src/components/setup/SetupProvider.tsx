'use client';

import { useEffect, useState, useCallback, useRef } from 'react';
import { Loader2, ArrowUpCircle, X } from 'lucide-react';
import { useSetupCheck } from '@/hooks/useSetupCheck';
import { SetupGate } from './SetupGate';
import { useAppStore } from '@/store/appStore';
import { useChannelStore } from '@/store/channelStore';

const BACKEND_TIMEOUT_MS = 3_000;

interface SetupProviderProps {
  children: React.ReactNode;
}

export function SetupProvider({ children }: SetupProviderProps) {
  const { isReady, isLoading, tools, missingRequired, error } =
    useSetupCheck();
  const [timedOut, setTimedOut] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [updateBannerDismissed, setUpdateBannerDismissed] = useState(false);
  const goUrl = useAppStore((s) => s.goUrl);
  const refreshConnectedChannels = useChannelStore((s) => s.refreshConnectedChannels);
  const refreshedRef = useRef(false);

  const updatableTools = tools.filter((t) => t.updateAvailable);

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

  // Refresh OAuth tokens for connected channels once the backend is ready
  useEffect(() => {
    if (!isReady || refreshedRef.current) return;
    refreshedRef.current = true;
    void refreshConnectedChannels(goUrl);
  }, [isReady, goUrl, refreshConnectedChannels]);

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

  // Backend offline after timeout — block the app until status is confirmed
  if (timedOut && isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-xs text-muted-foreground">Backend is starting up…</p>
        </div>
      </div>
    );
  }

  // Error (backend returned an error) — block the app, show recovery message
  if (error && tools.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-3 max-w-sm text-center">
          <p className="text-sm font-medium text-foreground">Backend offline</p>
          <p className="text-xs text-muted-foreground">
            Could not reach the AutoCut backend. Restart the app to try again.
          </p>
          <p className="text-xs text-muted-foreground/60">{error}</p>
        </div>
      </div>
    );
  }

  // Missing required tools — show SetupGate
  if (missingRequired.length > 0 && !dismissed) {
    return <SetupGate tools={tools} onComplete={handleComplete} />;
  }

  // All good — optionally show update banner
  const showUpdateBanner =
    updatableTools.length > 0 && !updateBannerDismissed;

  if (!showUpdateBanner) return <>{children}</>;

  return (
    <div className="relative h-full">
      <div className="sticky top-0 z-40 flex items-center gap-2 border-b border-amber-500/20 bg-amber-500/8 px-4 py-2 text-xs text-amber-400">
        <ArrowUpCircle className="h-3.5 w-3.5 shrink-0" />
        <span>
          Update available:{' '}
          {updatableTools
            .map((t) => `${t.name} ${t.latestVersion ?? ''}`.trim())
            .join(', ')}
          . Go to{' '}
          <a href="/settings" className="underline underline-offset-2">
            Settings → Tools
          </a>{' '}
          to update.
        </span>
        <button
          onClick={() => setUpdateBannerDismissed(true)}
          className="ml-auto shrink-0 opacity-60 hover:opacity-100"
          aria-label="Dismiss"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
      {children}
    </div>
  );
}
