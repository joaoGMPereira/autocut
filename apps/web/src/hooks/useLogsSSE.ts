// apps/web/src/hooks/useLogsSSE.ts
'use client';

import { useEffect } from 'react';
import { useLogStore } from '@/store/logStore';
import type { LogEntry, LogLevel } from '@autocut/shared';

export function useLogsSSE(goUrl: string) {
  const addEntry = useLogStore((s) => s.addEntry);

  useEffect(() => {
    // ── Console monkey-patch ──────────────────────────────────────────────────
    const origLog = console.log.bind(console);
    const origWarn = console.warn.bind(console);
    const origError = console.error.bind(console);
    const origDebug = console.debug.bind(console);

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

    console.log = makePatch('info', origLog);
    console.warn = makePatch('warn', origWarn);
    console.error = makePatch('error', origError);
    console.debug = makePatch('debug', origDebug);

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
      console.log = origLog;
      console.warn = origWarn;
      console.error = origError;
      console.debug = origDebug;
    };
  }, [goUrl, addEntry]);
}
