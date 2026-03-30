'use client';

import { Scissors } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useSetupStore } from '@/store/setupStore';
import { ToolRow } from './ToolRow';
import type { ToolStatus } from '@/types/setup';

interface SetupGateProps {
  tools: ToolStatus[];
  onComplete: () => void;
}

export function SetupGate({ tools, onComplete }: SetupGateProps) {
  const allRequiredInstalled = useSetupStore((s) => s.allRequiredInstalled);
  const isReady = allRequiredInstalled();

  return (
    <div className="flex h-full items-center justify-center">
      <div className="flex flex-col items-center gap-8 w-full max-w-[600px] px-4">
        {/* Header */}
        <div className="flex flex-col items-center gap-4">
          <Scissors className="h-14 w-14 text-blue-400" />
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            AutoCut Setup
          </h1>
          <p className="text-sm text-muted-foreground text-center max-w-[480px]">
            Some required tools need to be installed before you can use AutoCut.
          </p>
        </div>

        {/* Tool list */}
        <div className="w-full rounded-xl border border-border bg-card">
          {tools.map((tool, i) => (
            <div key={tool.name}>
              {i > 0 && <Separator />}
              <ToolRow tool={tool} />
            </div>
          ))}
        </div>

        {/* Continue */}
        <div className="flex flex-col items-center gap-3 w-full">
          <Button
            className="w-full h-11 text-sm font-semibold"
            disabled={!isReady}
            onClick={onComplete}
          >
            Continue
          </Button>
          {!isReady && (
            <p className="text-xs text-muted-foreground">
              Install all required tools to continue
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
