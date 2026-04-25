// apps/web/src/hooks/useLogsSSE.ts
'use client';

import { useEffect } from 'react';
import { useLogStore } from '@/store/logStore';
import type { LogEntry, LogLevel } from '@autocut/shared';

// Capture originals at module level — always the real browser functions,
// immune to React Strict Mode double-invocation stale capture.
const _origLog = typeof console !== 'undefined' ? console.log.bind(console) : undefined;
const _origWarn = typeof console !== 'undefined' ? console.warn.bind(console) : undefined;
const _origError = typeof console !== 'undefined' ? console.error.bind(console) : undefined;
const _origDebug = typeof console !== 'undefined' ? console.debug.bind(console) : undefined;

export function useLogsSSE(goUrl: string) {
  const addEntry = useLogStore((s) => s.addEntry);

  useEffect(() => {
    if (!_origLog || !_origWarn || !_origError || !_origDebug) return;

    function makePatch(level: LogLevel, orig: (...a: unknown[]) => void) {
      return (...args: unknown[]) => {
        orig(...args);
        addEntry({
          id: crypto.randomUUID(),
          ts: new Date().toISOString(),
          level,
          source: 'web',
          msg: args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a))).join(' '),
        });
      };
    }

    console.log = makePatch('info', _origLog);
    console.warn = makePatch('warn', _origWarn);
    console.error = makePatch('error', _origError);
    console.debug = makePatch('debug', _origDebug);

    // ── SSE connection with exponential backoff ────────────────────────────────
    let es: EventSource | null = null;
    let retryDelay = 1000;
    let retryTimeout: ReturnType<typeof setTimeout> | null = null;
    let unmounted = false;

    function connect() {
      es = new EventSource(`${goUrl}/api/logs/stream`);

      es.onmessage = (evt) => {
        try {
          const entry = JSON.parse(evt.data) as Partial<LogEntry>;
          if (entry.level && entry.source && entry.msg) {
            addEntry({
              id: entry.id ?? crypto.randomUUID(),
              ts: entry.ts ?? new Date().toISOString(),
              level: entry.level,
              source: entry.source,
              msg: entry.msg,
              attrs: entry.attrs,
            });
            retryDelay = 1000;
          }
        } catch {
          // ignore malformed events (e.g. keepalive ping)
        }
      };

      es.onerror = () => {
        es?.close();
        if (!unmounted) {
          retryTimeout = setTimeout(() => {
            retryDelay = Math.min(retryDelay * 2, 30000);
            connect();
          }, retryDelay);
        }
      };
    }

    connect();

    return () => {
      unmounted = true;
      if (retryTimeout) clearTimeout(retryTimeout);
      es?.close();
      console.log = _origLog!;
      console.warn = _origWarn!;
      console.error = _origError!;
      console.debug = _origDebug!;
    };
  }, [goUrl, addEntry]);
}
