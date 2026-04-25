# In-App Log Viewer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a floating log panel to the desktop app that streams Go `slog` output and frontend `console.*` calls in real time, with free-text search and error highlighting.

**Architecture:** A custom `slog.Handler` (`LogSink`) intercepts all Go log records, writes to stderr (unchanged), buffers the last 500 entries, and broadcasts to SSE listeners. A new `GET /api/logs/stream` endpoint replays the buffer then streams live. The frontend `useLogsSSE` hook connects to that stream and monkey-patches `console.*`, feeding everything into a Zustand `logStore`. A fixed `LogPanel` overlay renders the entries.

**Tech Stack:** Go `log/slog`, `net/http` SSE, Zustand, React, Tailwind, `@autocut/shared` types.

---

## File Map

| Action | Path | Purpose |
|--------|------|---------|
| Create | `server/internal/logsink/logsink.go` | Custom slog.Handler — buffer + broadcast |
| Create | `server/internal/logsink/logsink_test.go` | Unit tests for LogSink |
| Create | `server/internal/api/handlers/logs_handler.go` | HTTP handlers: SSE stream + history |
| Modify | `server/internal/api/router.go` | Add 2 new routes + LogsHandler param |
| Modify | `server/cmd/server/main.go` | Wire LogSink, create LogsHandler, pass to router |
| Modify | `packages/shared/src/types.ts` | Add `LogEntry` type |
| Create | `apps/web/src/store/logStore.ts` | Zustand store for log entries + panel state |
| Create | `apps/web/src/hooks/useLogsSSE.ts` | SSE hook + console monkey-patch |
| Create | `apps/web/src/components/LogPanel.tsx` | Floating log panel UI |
| Modify | `apps/web/src/components/app-sidebar.tsx` | Add Terminal icon that toggles panel |
| Modify | `apps/web/src/app/layout.tsx` | Mount LogPanel + useLogsSSE at root |

---

## Task 1: Go — `logsink` package

**Files:**
- Create: `server/internal/logsink/logsink.go`
- Create: `server/internal/logsink/logsink_test.go`

- [ ] **Step 1: Create `logsink.go`**

```go
// server/internal/logsink/logsink.go
package logsink

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

const bufCap = 500

// LogEntry is the JSON shape sent to the frontend.
type LogEntry struct {
	ID     string                 `json:"id"`
	Ts     string                 `json:"ts"`
	Level  string                 `json:"level"`
	Source string                 `json:"source"`
	Msg    string                 `json:"msg"`
	Attrs  map[string]interface{} `json:"attrs,omitempty"`
}

// sharedState is held by pointer so WithAttrs/WithGroup clones share it.
type sharedState struct {
	mu      sync.Mutex
	buf     []LogEntry
	clients []chan LogEntry
	seq     uint64
}

// LogSink is a slog.Handler that tees to an inner handler, buffers entries,
// and broadcasts to registered SSE listeners.
type LogSink struct {
	inner  slog.Handler
	shared *sharedState
}

// New creates a LogSink that writes text logs to w (stderr in production).
func New(w io.Writer) *LogSink {
	return &LogSink{
		inner: slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}),
		shared: &sharedState{
			buf: make([]LogEntry, 0, bufCap),
		},
	}
}

func (s *LogSink) Enabled(ctx context.Context, level slog.Level) bool {
	return s.inner.Enabled(ctx, level)
}

func (s *LogSink) Handle(ctx context.Context, r slog.Record) error {
	// Always write to stderr via inner handler; ignore its error.
	_ = s.inner.Handle(ctx, r)

	attrs := make(map[string]interface{})
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	if len(attrs) == 0 {
		attrs = nil
	}

	s.shared.mu.Lock()
	s.shared.seq++
	entry := LogEntry{
		ID:     fmt.Sprintf("%d", s.shared.seq),
		Ts:     r.Time.UTC().Format(time.RFC3339Nano),
		Level:  levelName(r.Level),
		Source: "go",
		Msg:    r.Message,
		Attrs:  attrs,
	}
	if len(s.shared.buf) >= bufCap {
		s.shared.buf = append(s.shared.buf[1:], entry)
	} else {
		s.shared.buf = append(s.shared.buf, entry)
	}
	clients := make([]chan LogEntry, len(s.shared.clients))
	copy(clients, s.shared.clients)
	s.shared.mu.Unlock()

	for _, ch := range clients {
		select {
		case ch <- entry:
		default:
		}
	}
	return nil
}

func (s *LogSink) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogSink{inner: s.inner.WithAttrs(attrs), shared: s.shared}
}

func (s *LogSink) WithGroup(name string) slog.Handler {
	return &LogSink{inner: s.inner.WithGroup(name), shared: s.shared}
}

// Register adds a listener. Returns a receive channel pre-loaded with the
// replay buffer and a cancel func that must be called on disconnect.
func (s *LogSink) Register() (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, 128)

	s.shared.mu.Lock()
	for _, e := range s.shared.buf {
		select {
		case ch <- e:
		default:
		}
	}
	s.shared.clients = append(s.shared.clients, ch)
	s.shared.mu.Unlock()

	cancel := func() {
		s.shared.mu.Lock()
		defer s.shared.mu.Unlock()
		for i, c := range s.shared.clients {
			if c == ch {
				s.shared.clients = append(s.shared.clients[:i], s.shared.clients[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, cancel
}

// History returns a snapshot of the ring buffer.
func (s *LogSink) History() []LogEntry {
	s.shared.mu.Lock()
	defer s.shared.mu.Unlock()
	out := make([]LogEntry, len(s.shared.buf))
	copy(out, s.shared.buf)
	return out
}

func levelName(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "error"
	case l >= slog.LevelWarn:
		return "warn"
	case l >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}
```

- [ ] **Step 2: Create `logsink_test.go`**

```go
// server/internal/logsink/logsink_test.go
package logsink

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func newTestSink() *LogSink {
	return New(io.Discard)
}

func makeRecord(msg string, level slog.Level) slog.Record {
	return slog.NewRecord(time.Now(), level, msg, 0)
}

func TestHandle_BuffersEntry(t *testing.T) {
	s := newTestSink()
	r := makeRecord("hello", slog.LevelInfo)
	_ = s.Handle(context.Background(), r)

	hist := s.History()
	if len(hist) != 1 {
		t.Fatalf("want 1 entry, got %d", len(hist))
	}
	if hist[0].Msg != "hello" {
		t.Errorf("want msg=hello, got %q", hist[0].Msg)
	}
	if hist[0].Level != "info" {
		t.Errorf("want level=info, got %q", hist[0].Level)
	}
	if hist[0].Source != "go" {
		t.Errorf("want source=go, got %q", hist[0].Source)
	}
}

func TestHandle_BroadcastsToListener(t *testing.T) {
	s := newTestSink()
	ch, cancel := s.Register()
	defer cancel()

	r := makeRecord("broadcast", slog.LevelWarn)
	_ = s.Handle(context.Background(), r)

	select {
	case entry := <-ch:
		if entry.Msg != "broadcast" {
			t.Errorf("want msg=broadcast, got %q", entry.Msg)
		}
		if entry.Level != "warn" {
			t.Errorf("want level=warn, got %q", entry.Level)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestHandle_RingBuffer_Evicts(t *testing.T) {
	s := newTestSink()
	for i := 0; i < bufCap+10; i++ {
		r := makeRecord("msg", slog.LevelInfo)
		_ = s.Handle(context.Background(), r)
	}
	hist := s.History()
	if len(hist) != bufCap {
		t.Fatalf("want buf len=%d, got %d", bufCap, len(hist))
	}
}

func TestRegister_ReplaysBuf(t *testing.T) {
	s := newTestSink()
	for _, msg := range []string{"a", "b", "c"} {
		r := makeRecord(msg, slog.LevelInfo)
		_ = s.Handle(context.Background(), r)
	}

	ch, cancel := s.Register()
	defer cancel()

	var got []string
	for i := 0; i < 3; i++ {
		select {
		case e := <-ch:
			got = append(got, e.Msg)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout at entry %d", i)
		}
	}
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("unexpected replay: %v", got)
	}
}

func TestHistory_ReturnsCopy(t *testing.T) {
	s := newTestSink()
	r := makeRecord("x", slog.LevelInfo)
	_ = s.Handle(context.Background(), r)

	h1 := s.History()
	h1[0].Msg = "mutated"
	h2 := s.History()
	if h2[0].Msg == "mutated" {
		t.Error("History should return a copy, not a reference")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut/server
go test ./internal/logsink/...
```

Expected: `ok github.com/joaoGMPereira/autocut/server/internal/logsink`

- [ ] **Step 4: Commit**

```bash
git add server/internal/logsink/
git commit -m "feat(logsink): add custom slog.Handler with ring buffer and SSE broadcast"
```

---

## Task 2: Go — Logs HTTP handler

**Files:**
- Create: `server/internal/api/handlers/logs_handler.go`

- [ ] **Step 1: Create `logs_handler.go`**

```go
// server/internal/api/handlers/logs_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/logsink"
)

// LogsHandler serves the in-app log stream.
type LogsHandler struct {
	sink *logsink.LogSink
	log  *slog.Logger
}

// NewLogsHandler creates a LogsHandler backed by the given LogSink.
func NewLogsHandler(sink *logsink.LogSink) *LogsHandler {
	return &LogsHandler{
		sink: sink,
		log:  slog.With("component", "handler.logs"),
	}
}

// GetHistory returns the current ring buffer as a JSON array.
func (h *LogsHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	entries := h.sink.History()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		h.log.Error("failed to encode log history", "err", err)
	}
}

// GetStream opens an SSE connection that replays the buffer then streams live.
func (h *LogsHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch, cancel := h.sink.Register()
	defer cancel()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case entry, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()

		case <-ticker.C:
			if _, err := fmt.Fprintf(w, "data: {\"ping\":true}\n\n"); err != nil {
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut/server
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add server/internal/api/handlers/logs_handler.go
git commit -m "feat(logs): add LogsHandler for SSE stream and history endpoints"
```

---

## Task 3: Go — Wire into router and main.go

**Files:**
- Modify: `server/internal/api/router.go`
- Modify: `server/cmd/server/main.go`

- [ ] **Step 1: Add LogsHandler param and routes to `router.go`**

In `server/internal/api/router.go`, add `lh *handlers.LogsHandler` as the last parameter to `NewRouter` and register the two new routes:

Find the existing `NewRouter` signature:
```go
func NewRouter(
	ph *handlers.PipelineHandler,
	...
	cch *handlers.ChannelConfigHandler,
) http.Handler {
```

Change it to:
```go
func NewRouter(
	ph *handlers.PipelineHandler,
	dh *handlers.DownloadHandler,
	sh *handlers.SetupHandler,
	sth *handlers.StatsHandler,
	uhh *handlers.URLHistoryHandler,
	seth *handlers.SettingsHandler,
	ch *handlers.ChannelHandler,
	oh *handlers.OAuthHandler,
	olh *handlers.OllamaHandler,
	mlh *handlers.MediaLibraryHandler,
	pvh *handlers.PreviewHandler,
	th *handlers.ThumbnailHandler,
	mh *handlers.MetadataHandler,
	qh *handlers.QueueHandler,
	cch *handlers.ChannelConfigHandler,
	lh *handlers.LogsHandler,
) http.Handler {
```

Then add the two routes just before the health check block:
```go
	// In-app log viewer endpoints
	mux.HandleFunc("GET /api/logs/stream", lh.GetStream)
	mux.HandleFunc("GET /api/logs/history", lh.GetHistory)
```

- [ ] **Step 2: Wire `LogSink` in `main.go`**

In `server/cmd/server/main.go`, replace the slog setup block:
```go
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
```

With:
```go
	sink := logsink.New(os.Stderr)
	slog.SetDefault(slog.New(sink))
```

Add the import at the top of main.go imports:
```go
	"github.com/joaoGMPereira/autocut/server/internal/logsink"
```

Then, after `channelConfigHandler := handlers.NewChannelConfigHandler(channelCfgRepo)`, add:
```go
	logsHandler := handlers.NewLogsHandler(sink)
```

And add `logsHandler` as the last argument to `api.NewRouter(...)`:
```go
	router := api.NewRouter(
		pipelineHandler,
		downloadHandler,
		setupHandler,
		statsHandler,
		urlHistoryHandler,
		settingsHandler,
		channelHandler,
		oauthHandler,
		ollamaHandler,
		mediaLibraryHandler,
		previewHandler,
		thumbnailHandler,
		metadataHandler,
		queueHandler,
		channelConfigHandler,
		logsHandler,
	)
```

- [ ] **Step 3: Build and run tests**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut/server
go build ./...
go test ./...
```

Expected: all pass, no compile errors.

- [ ] **Step 4: Commit**

```bash
git add server/internal/api/router.go server/cmd/server/main.go
git commit -m "feat(logs): wire LogSink into router and main — /api/logs/stream live"
```

---

## Task 4: Frontend — Shared `LogEntry` type

**Files:**
- Modify: `packages/shared/src/types.ts`

- [ ] **Step 1: Append `LogEntry` to types.ts**

Open `packages/shared/src/types.ts` and add at the bottom:

```ts
// ─── Log Viewer ───────────────────────────────────────────────────────────────

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';
export type LogSource = 'go' | 'web';

export interface LogEntry {
  id: string;
  ts: string;
  level: LogLevel;
  source: LogSource;
  msg: string;
  attrs?: Record<string, unknown>;
}
```

- [ ] **Step 2: Verify TypeScript build**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut
pnpm --filter @autocut/shared build 2>/dev/null || pnpm --filter @autocut/shared tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add packages/shared/src/types.ts
git commit -m "feat(shared): add LogEntry type for in-app log viewer"
```

---

## Task 5: Frontend — `logStore`

**Files:**
- Create: `apps/web/src/store/logStore.ts`

- [ ] **Step 1: Create `logStore.ts`**

```ts
// apps/web/src/store/logStore.ts
import { create } from 'zustand';
import type { LogEntry } from '@autocut/shared';

const MAX_ENTRIES = 1000;

interface LogState {
  entries: LogEntry[];
  search: string;
  open: boolean;
  addEntry: (entry: LogEntry) => void;
  clear: () => void;
  setSearch: (search: string) => void;
  setOpen: (open: boolean) => void;
  toggleOpen: () => void;
}

export const useLogStore = create<LogState>((set) => ({
  entries: [],
  search: '',
  open: false,

  addEntry: (entry) =>
    set((state) => {
      const next = [...state.entries, entry];
      return {
        entries: next.length > MAX_ENTRIES ? next.slice(next.length - MAX_ENTRIES) : next,
      };
    }),

  clear: () => set({ entries: [] }),
  setSearch: (search) => set({ search }),
  setOpen: (open) => set({ open }),
  toggleOpen: () => set((state) => ({ open: !state.open })),
}));
```

- [ ] **Step 2: TypeScript check**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut
pnpm --filter @autocut/web tsc --noEmit
```

Expected: no errors relating to `logStore.ts`.

- [ ] **Step 3: Commit**

```bash
git add apps/web/src/store/logStore.ts
git commit -m "feat(logStore): add Zustand store for log entries and panel state"
```

---

## Task 6: Frontend — `useLogsSSE` hook

**Files:**
- Create: `apps/web/src/hooks/useLogsSSE.ts`

- [ ] **Step 1: Create `useLogsSSE.ts`**

```ts
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
```

- [ ] **Step 2: TypeScript check**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut
pnpm --filter @autocut/web tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add apps/web/src/hooks/useLogsSSE.ts
git commit -m "feat(useLogsSSE): SSE hook with console monkey-patch and reconnect backoff"
```

---

## Task 7: Frontend — `LogPanel` component

**Files:**
- Create: `apps/web/src/components/LogPanel.tsx`

- [ ] **Step 1: Create `LogPanel.tsx`**

```tsx
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
```

- [ ] **Step 2: TypeScript check**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut
pnpm --filter @autocut/web tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add apps/web/src/components/LogPanel.tsx
git commit -m "feat(LogPanel): floating log panel with search, auto-scroll, and keyboard shortcut"
```

---

## Task 8: Frontend — Wire into layout and sidebar

**Files:**
- Modify: `apps/web/src/app/layout.tsx`
- Modify: `apps/web/src/components/app-sidebar.tsx`

- [ ] **Step 1: Add `LogsProvider` client component**

The `layout.tsx` is a Server Component, so we can't call hooks directly in it. Create a small client wrapper:

```tsx
// apps/web/src/components/LogsProvider.tsx
'use client';

import { useAppStore } from '@/store/appStore';
import { useLogsSSE } from '@/hooks/useLogsSSE';
import { LogPanel } from '@/components/LogPanel';

export function LogsProvider() {
  const goUrl = useAppStore((s) => s.goUrl);
  useLogsSSE(goUrl);
  return <LogPanel />;
}
```

- [ ] **Step 2: Mount `LogsProvider` in `layout.tsx`**

In `apps/web/src/app/layout.tsx`, add the import:
```ts
import { LogsProvider } from '@/components/LogsProvider';
```

And mount it inside `<UpdateStatusProvider>`, after `</div>` (the DispatcherBar div) and before `</UpdateStatusProvider>`:

Current end of body:
```tsx
            <div className="fixed bottom-4 right-4 z-50 w-80">
              <DispatcherBar />
            </div>
          </UpdateStatusProvider>
```

Change to:
```tsx
            <div className="fixed bottom-4 right-4 z-50 w-80">
              <DispatcherBar />
            </div>
            <LogsProvider />
          </UpdateStatusProvider>
```

- [ ] **Step 3: Add Terminal icon to sidebar**

In `apps/web/src/components/app-sidebar.tsx`, add the `Terminal` import:
```ts
import {
  Home,
  Play,
  Upload,
  Users,
  Settings,
  Terminal,
} from 'lucide-react';
```

Add the `useLogStore` import:
```ts
import { useLogStore } from '@/store/logStore';
```

Inside `AppSidebar`, add the store access right after the `usePathname` line:
```ts
  const toggleOpen = useLogStore((s) => s.toggleOpen);
  const isLogOpen = useLogStore((s) => s.open);
```

At the bottom of the `<nav>` (after the existing nav items loop, before `</nav>`), add:
```tsx
        {/* Log viewer toggle — pinned at bottom */}
        <div className="mt-auto pt-1">
          <Tooltip delayDuration={200}>
            <TooltipTrigger asChild>
              <button
                onClick={toggleOpen}
                className={cn(
                  'flex items-center justify-center h-9 w-9 rounded-md transition-colors',
                  'hover:bg-surface hover:text-brand',
                  isLogOpen ? 'bg-surface text-brand' : 'text-subtle',
                )}
              >
                <Terminal className="h-4 w-4" />
                <span className="sr-only">Logs</span>
              </button>
            </TooltipTrigger>
            <TooltipContent side="right" sideOffset={8}>
              Logs
            </TooltipContent>
          </Tooltip>
        </div>
```

- [ ] **Step 4: TypeScript check**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut
pnpm --filter @autocut/web tsc --noEmit
```

Expected: no errors.

- [ ] **Step 5: Start the dev server and verify manually**

```bash
cd /Users/joaogabriel/Projects/YoutubeProjects/AutoCut
make run
```

Open the app, click the Terminal icon in the sidebar. The panel should open at the bottom. Go backend logs should appear. Press `Cmd+Shift+L` to toggle. Type in the search box to filter. Click "clear" to clear.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/LogsProvider.tsx apps/web/src/app/layout.tsx apps/web/src/components/app-sidebar.tsx
git commit -m "feat(ui): wire LogPanel into layout and add Terminal toggle to sidebar"
```
