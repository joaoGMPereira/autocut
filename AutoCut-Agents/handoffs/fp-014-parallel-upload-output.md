---
feature: FP-014-ParallelUpload
stage: output
scope: full
status: approved
files_created:
  - AutoCut/server/internal/uploader/strategy.go
files_modified:
  - AutoCut/server/internal/api/handlers/upload_handler.go
  - AutoCut/apps/web/src/components/settings/AppSettingsSection.tsx
---

## Summary

FP-014 Parallel Upload Strategy — backend helper + frontend UI.

### Backend

- Created `strategy.go` in `internal/uploader` with `UploadStrategy` struct, semaphore-based
  concurrency control, and `NewUploadStrategy(strategy, count)` constructor.
  Clamps count to [1, 5]; sequential vs parallel controlled by strategy string.
- Added `// TODO: Read upload_strategy + upload_parallel_count from settings and use UploadStrategy
  for batch uploads` comment in `upload_handler.go` above `PostUpload`.
- No new API endpoints needed: existing `GET /api/settings` + `PUT /api/settings` handle the two
  new keys (`upload_strategy`, `upload_parallel_count`) transparently via the generic key-value store.

### Frontend

- `AppSettingsSection.tsx`:
  - Added `Label` import from `@/components/ui/label`.
  - Added `uploadStrategy` and `uploadParallelCount` local state (synced from settings on load).
  - Extended `handleSave` to call `saveSetting` for both new keys after `saveAllSettings`.
  - Added "Upload Strategy" section inside the card with:
    - A native `<button role="switch">` toggle (no Switch shadcn component available in this codebase).
    - A native `<input type="range">` slider (1–5) that is conditionally visible when parallel is enabled.
    - Quota advice hint text.

### Build verification

- `CGO_ENABLED=0 go build ./...` — clean (no output).
- `pnpm tsc --noEmit` — clean (no output).
