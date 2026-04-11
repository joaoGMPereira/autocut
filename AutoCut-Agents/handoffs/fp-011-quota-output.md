---
feature: FP-011-QuotaMonitor
stage: output
scope: full
status: approved
files_created:
  - AutoCut/server/internal/api/handlers/quota_handler.go
  - AutoCut/apps/web/src/store/quotaStore.ts
  - AutoCut/apps/web/src/app/quota/page.tsx
files_modified:
  - AutoCut/server/internal/api/handlers/stats_handler.go
  - AutoCut/server/internal/api/router.go
  - AutoCut/server/cmd/server/main.go
  - AutoCut/server/cmd/autocut/main.go
  - AutoCut/apps/web/src/components/app-sidebar.tsx
---

## Summary

FP-011 Quota Monitoring Dashboard is fully implemented, backend and frontend.

### Backend

**stats_handler.go** — Added `QuotaBreakdown` struct and a 7th query in `GetStats` that
sums `upload_count`, `thumbnail_count`, and `other_api_calls` from `quota_usage` for
today (PT). The breakdown is embedded in `StatsResponse` as `quota_breakdown`.

**quota_handler.go** — New handler with two methods:
- `GetQuotaHistory` → `GET /api/quota/history`: groups `quota_usage` by date (last 30),
  sums across all secrets, returns `{ history: [...] }`.
- `PostQuotaReset` → `POST /api/quota/reset`: zeroes all counter columns for today's
  rows and bumps `updated_at`.

**router.go** — Added `quotaH *handlers.QuotaHandler` parameter and registered the two
new routes under `GET /api/quota/history` and `POST /api/quota/reset`.

**cmd/server/main.go + cmd/autocut/main.go** — Both entrypoints wired with
`quotaH := handlers.NewQuotaHandler(db)` and passed as the final argument to
`api.NewRouter(...)`. Build verified clean (`CGO_ENABLED=0 go build ./...`).

### Frontend

**quotaStore.ts** — Zustand store with `breakdown`, `history`, `loading`, and three
async actions: `fetchBreakdown` (GET /api/stats), `fetchHistory` (GET /api/quota/history),
`resetQuota` (POST /api/quota/reset).

**app/quota/page.tsx** — `/quota` page with:
- Anton-font title "QUOTA MONITOR"
- Four `QuotaBar` progress bars (Uploads / Updates / Reads / Total Units) with green /
  yellow / red colour coding at <60% / 60-80% / >80%.
- `MiniBarChart` CSS-only bar chart showing the last 7 days from history data.
- "Reset Today's Count" button with `confirm()` guard.
- Auto-fetches both breakdown and history on mount.

**app-sidebar.tsx** — `Gauge` icon imported from lucide-react; "Quota" nav item added
between Queue and Channels pointing to `/quota`.

TypeScript check passed (`pnpm tsc --noEmit` — no output = no errors).

### Notes

- A second entrypoint `cmd/autocut/main.go` (parallel to `cmd/server/main.go`) also
  required the same `quotaH` wiring — updated accordingly.
- `UploadsLimit` in the breakdown is set to `1600` (one upload cost unit) per spec;
  this is the per-upload cost constant, not a count-based daily cap — adjust if the
  intent is a daily upload count limit instead.
