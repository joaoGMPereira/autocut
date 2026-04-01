'use client';

import { useEffect, useRef } from 'react';
import { ScrollArea } from '@/components/ui/scroll-area';

interface LogViewerProps {
  logs: string[];
}

export function LogViewer({ logs }: LogViewerProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs.length]);

  return (
    <ScrollArea className="h-40 rounded-lg bg-[#0D0D14] border border-border mt-3">
      <div className="p-3 font-mono text-xs space-y-0.5">
        {logs.length === 0 ? (
          <span className="text-[#5C5C80]">Waiting for output...</span>
        ) : (
          logs.map((line, i) => (
            <div key={i} className="text-[#A0A0B8] leading-relaxed">
              {line}
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </ScrollArea>
  );
}
