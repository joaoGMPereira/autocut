# Implementation Plan: Download Pipeline — Validation & Hardening

**Branch**: `002-download-pipeline-validation` | **Date**: 2026-04-12 | **Spec**: specs/002-download-pipeline-validation/spec.md
**Input**: Feature specification from `/specs/002-download-pipeline-validation/spec.md`

## Summary

Fix six broken behaviours in the existing download pipeline — no intermediate progress, missing speed/ETA fields, un-persisted video path and title, uncancellable goroutine, empty TODO stub files, and absent URL validation — and add a typed phase output contract so future pipeline phases can safely depend on download's guarantees. One DB migration (v10) is required to add `video_title` and `duration_sec` columns.

## Technical Context

**Language/Version**: Go 1.23 (backend), TypeScript 5.8 (frontend)
**Primary Dependencies**: modernc.org/sqlite (CGO_ENABLED=0), net/http stdlib, log/slog, yt-dlp subprocess, Next.js 16, React, Tailwind CSS, Zustand
**Storage**: SQLite WAL — `video_path` exists in v9 schema; `video_title` and `duration_sec` absent → migration v10 required (see research.md §2)
**Testing**: `CGO_ENABLED=0 go test ./...` (backend), `pnpm tsc --noEmit` + `pnpm test` (frontend)
**Target Platform**: macOS/Linux server + web browser (Electron desktop app)
**Project Type**: desktop-app + web-service
**Performance Goals**: `phase_progress` events ≤ 5 s apart; subprocess terminated ≤ 3 s after cancel
**Constraints**: CGO_ENABLED=0 always; no net/http in business logic; no standalone test pages

## Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| Business logic in `server/internal/` | PASS | pipeline.Service owns all logic |
| No net/http in business logic | PASS | Service touches only hub.SSEHub + database |
| SSE Hub: one hub, run ID as key | PASS | `fmt.Sprintf("%d", id)` throughout |
| Canonical SSE event types | PARTIAL — phaseProgressPayload missing speed_kbs/eta_sec | Resolved by this feature |
| No standalone test pages | PASS | /download-test deleted on this branch |
| CGO_ENABLED=0 | PASS | modernc.org/sqlite driver |
| Pipeline integration order | PASS | service→handler→router→main unchanged |
| FE integrates into real pipeline UI | PASS | DownloadInfoCard lives inside ContextPanel |

No HIGH/CRITICAL violations. All gaps are implementation-level fixes, not architectural violations.

## Project Structure

### Documentation (this feature)

```text
specs/002-download-pipeline-validation/
├── plan.md              # This file
├── research.md          # Phase 0 — resolved unknowns
├── data-model.md        # Phase 1 — entities and state transitions
├── quickstart.md        # Phase 1 — local test recipe
└── contracts/
    └── download-phase.md   # Phase 1 — typed phase contract
```

### Source Code (affected paths)

```text
server/internal/
├── database/
│   ├── migrations.go              # + migrateV10: ADD COLUMN video_title, duration_sec
│   ├── models.go                  # + VideoTitle string, DurationSec int64 on PipelineRun
│   └── pipeline_run_repo.go       # + SetDownloadResult(id, path, title, durSec)
├── downloader/
│   ├── executor.go                # + StreamingExecutor: Run(ctx, name, args, onLine) ([]byte, error)
│   ├── youtube.go                 # Refactor DownloadWithOptions: context + streaming progress
│   └── types.go                   # + SpeedKbs, EtaSec to progress line parser output
├── pipeline/
│   ├── service.go                 # Fix runDownload: ctx propagation, progress, persist result
│   ├── contracts.go               # NEW: PhaseContract type + download contract declaration
│   ├── state_machine.go           # REPLACE stub: ValidatePhaseInputs helper
│   ├── types.go                   # REPLACE stub: RunState / PhaseID string constants
│   └── [executor/sse/interfaces/repo/adapters].go  # REPLACE stubs or DELETE if unused
apps/web/src/
├── types/pipeline.ts              # + speed_kbs?, eta_sec? to PhaseProgressEvent
├── store/pipelineStore.ts         # wire speed_kbs, eta_sec into downloadProgress state
└── components/pipeline/
    └── DownloadInfoCard.tsx       # render speed + ETA when present
scripts/validate/
├── validate-02.sh                 # NEW: BE validation (progress events, video_path, cancel)
└── validate-02-fe.sh              # NEW: FE validation (DownloadInfoCard, no test pages)
```

**Structure Decision**: Single monorepo. Backend in `server/internal/`, frontend in `apps/web/`. No new top-level projects.

## Complexity Tracking

No constitution violations requiring justification.
