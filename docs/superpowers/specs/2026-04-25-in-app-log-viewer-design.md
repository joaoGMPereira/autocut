# In-App Log Viewer — Design Spec

**Date:** 2026-04-25  
**Status:** Approved

## Overview

A floating log panel inside the desktop app that shows all Go backend logs (`slog`) and frontend logs (`console.*`) in real time, with free-text search and error highlighting. Eliminates the need to watch the terminal during debugging.

---

## Architecture

### Backend (Go)

**`server/internal/logsink/logsink.go`** — new package  
Custom `slog.Handler` that:
1. Delegates every record to the existing `TextHandler` (stderr stays intact)
2. Appends the entry to an in-memory ring buffer (cap 500, thread-safe)
3. Broadcasts to all active SSE listeners

```go
type LogSink struct {
    inner   slog.Handler         // existing TextHandler→stderr
    mu      sync.Mutex
    buf     []LogEntry           // ring buffer, cap 500
    clients []chan LogEntry       // active SSE listeners
}
```

**`LogEntry` (JSON shape sent over SSE and REST):**
```json
{
  "id":     "nanoid",
  "ts":     "2026-04-25T12:03:41Z",
  "level":  "info",             // debug | info | warn | error
  "source": "go",
  "msg":    "server started",
  "attrs":  { "port": 4071 }   // slog key-value pairs, may be empty
}
```

**New endpoints (added to `router.go`):**
- `GET /api/logs/stream` — SSE stream. On connect: replays ring buffer, then streams live entries.
- `GET /api/logs/history` — returns ring buffer as JSON array (fallback / reconnect use).

**Wiring in `main.go`:**  
Replace `slog.SetDefault(slog.New(slog.NewTextHandler(...)))` with:
```go
sink := logsink.New(os.Stderr)
slog.SetDefault(slog.New(sink))
```
Pass `sink` into the router so the new handlers can access it.

---

### Frontend (TypeScript / React)

**`packages/shared/src/index.ts`** — add shared type:
```ts
export type LogEntry = {
  id: string
  ts: string
  level: 'debug' | 'info' | 'warn' | 'error'
  source: 'go' | 'web'
  msg: string
  attrs?: Record<string, unknown>
}
```

**`apps/web/src/store/logStore.ts`** — Zustand store:
- Holds up to 1000 `LogEntry` items (FIFO eviction)
- Actions: `addEntry`, `clear`, `setSearch`, `setOpen`
- Derived: `filtered` (substring match on `msg` + stringified `attrs`)

**`apps/web/src/hooks/useLogsSSE.ts`** — SSE hook:
- Connects to `${NEXT_PUBLIC_GO_URL}/api/logs/stream`
- Parses each event, calls `logStore.addEntry`
- On disconnect: exponential backoff reconnect (1s → 2s → 4s, max 30s)
- On mount: also monkey-patches `console.log/warn/error/debug` to inject `source: 'web'` entries into the store. Restores originals on unmount.

**`apps/web/src/components/LogPanel.tsx`** — UI:
- Fixed overlay: `position: fixed, bottom: 0, left: 56px, right: 0, height: 40vh, z-index: 40`
- Header: terminal icon + "Logs", search input, close button
- Body: scrollable list of log lines (font mono, `text-xs`)
  - `[go]` tag: `text-brand`
  - `[web]` tag: `text-purple-400`
  - `error` level: `text-destructive`
  - `warn` level: `text-yellow-400`
- Footer: "auto-scroll" toggle (default on) + "clear" button
- Keyboard shortcut: `Cmd+Shift+L` (Mac) / `Ctrl+Shift+L` (Windows) — toggles panel

**`apps/web/src/components/app-sidebar.tsx`** — add `Terminal` icon nav item at the bottom, below `Settings`, that toggles `logStore.open`.

**`apps/web/src/app/layout.tsx`** — mount `<LogPanel />` and `useLogsSSE` once at root level (so logs accumulate regardless of current page).

---

## Data Flow

```
slog.Info("x", "k", v)
  → LogSink.Handle()
    → TextHandler (stderr)
    → append to ring buffer
    → broadcast to SSE clients
      → GET /api/logs/stream
        → useLogsSSE hook
          → logStore.addEntry()
            → LogPanel renders line

console.error("oops")
  → monkey-patched wrapper
    → original console.error (browser devtools)
    → logStore.addEntry({ source: "web", level: "error", msg: "oops" })
      → LogPanel renders line
```

---

## Error Handling

- SSE disconnects: reconnect with exponential backoff, no user action needed.
- Ring buffer full: FIFO — oldest entry evicted silently (cap 500 backend, 1000 frontend).
- `logsink.Handle()` never panics — errors from inner handler are silently swallowed (log-on-log infinite loop prevention).
- `GET /api/logs/history` used by SSE handler for replay; not exposed to the user directly.

---

## Out of Scope

- Persisting logs to disk / across restarts
- Log level filter toggles (INFO/WARN/ERROR buttons) — only search + error highlight
- Exporting logs to file
- Backend DEBUG level by default (stays INFO)
