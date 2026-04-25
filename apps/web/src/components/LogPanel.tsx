// apps/web/src/components/LogPanel.tsx
'use client';

import { useState, useEffect, useRef } from 'react';
import { Terminal, X, Search } from 'lucide-react';
import { useLogStore } from '@/store/logStore';
import type { LogEntry } from '@autocut/shared';
import { cn } from '@/lib/utils';

function LogLine({ entry }: { entry: LogEntry }) {
  const time = new Date(entry.ts).toLocaleTimeString('en', { hour12: false });
  const hasAttrs = entry.attrs && Object.keys(entry.attrs).length > 0;

  return (
    <div className="flex gap-2 py-0.5 min-w-0 leading-relaxed">
      <span className="text-caption shrink-0 tabular-nums">{time}</span>
      <span
        className={cn(
          'shrink-0 font-semibold',
          entry.source === 'go' ? 'text-brand' : 'text-purple-400',
        )}
      >
        [{entry.source}]
      </span>
      <span
        className={cn(
          'break-all',
          entry.level === 'error' && 'text-destructive',
          entry.level === 'warn' && 'text-yellow-400',
          entry.level !== 'error' && entry.level !== 'warn' && 'text-prose',
        )}
      >
        {entry.msg}
        {hasAttrs && (
          <span className="text-caption ml-2">{JSON.stringify(entry.attrs)}</span>
        )}
      </span>
    </div>
  );
}

export function LogPanel() {
  const open = useLogStore((s) => s.open);
  const entries = useLogStore((s) => s.entries);
  const search = useLogStore((s) => s.search);
  const setSearch = useLogStore((s) => s.setSearch);
  const setOpen = useLogStore((s) => s.setOpen);
  const clear = useLogStore((s) => s.clear);
  const toggleOpen = useLogStore((s) => s.toggleOpen);

  const [autoScroll, setAutoScroll] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  const filtered = search
    ? entries.filter(
        (e) =>
          e.msg.toLowerCase().includes(search.toLowerCase()) ||
          (e.attrs &&
            JSON.stringify(e.attrs).toLowerCase().includes(search.toLowerCase())),
      )
    : entries;

  // Auto-scroll to bottom when new entries arrive
  useEffect(() => {
    if (autoScroll && open) {
      bottomRef.current?.scrollIntoView({ behavior: 'instant' });
    }
  }, [filtered.length, autoScroll, open]);

  // Keyboard shortcut: Cmd+Shift+L / Ctrl+Shift+L
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 'L') {
        e.preventDefault();
        toggleOpen();
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [toggleOpen]);

  if (!open) return null;

  return (
    <div className="fixed bottom-0 left-14 right-0 h-[40vh] z-40 border-t border-border bg-background flex flex-col shadow-2xl">
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border shrink-0">
        <Terminal className="h-3.5 w-3.5 text-caption" />
        <span className="text-xs font-semibold text-prose">Logs</span>
        <div className="flex-1 relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-caption pointer-events-none" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="filter..."
            className="w-full bg-surface-inset text-xs pl-6 pr-2 py-0.5 rounded border border-border outline-none focus:border-brand font-mono text-prose placeholder:text-caption"
          />
        </div>
        <button
          onClick={() => setOpen(false)}
          className="text-caption hover:text-prose transition-colors"
          aria-label="Close log panel"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Body */}
      <div ref={bodyRef} className="flex-1 overflow-y-auto px-3 py-1 font-mono text-xs">
        {filtered.length === 0 ? (
          <p className="text-caption py-2">No log entries.</p>
        ) : (
          filtered.map((e) => <LogLine key={e.id} entry={e} />)
        )}
        <div ref={bottomRef} />
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between px-3 py-1 border-t border-border shrink-0">
        <label className="flex items-center gap-1.5 text-xs text-caption cursor-pointer select-none">
          <input
            type="checkbox"
            checked={autoScroll}
            onChange={(e) => setAutoScroll(e.target.checked)}
            className="w-3 h-3 accent-brand"
          />
          auto-scroll
        </label>
        <span className="text-xs text-caption">{filtered.length} entries</span>
        <button
          onClick={clear}
          className="text-xs text-caption hover:text-prose transition-colors"
        >
          clear
        </button>
      </div>
    </div>
  );
}
