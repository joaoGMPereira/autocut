---
feature: FP-007-ThumbnailTemplate
stage: output
scope: frontend
status: approved
files_created:
  - AutoCut/apps/web/src/store/thumbnailStore.ts
  - AutoCut/apps/web/src/components/thumbnail/BackgroundsGallery.tsx
files_modified:
  - AutoCut/apps/web/src/app/thumbnail/page.tsx
---

## Summary

FP-007 frontend implementation — Template Thumbnail Strategy with backgrounds gallery.

### thumbnailStore.ts

Zustand store (`useThumbnailStore`) managing background images per channel:

- `fetchBackgrounds(channelId)` — GET `/api/channels/:id/backgrounds`
- `uploadBackground(channelId, file, name)` — POST multipart form to `/api/channels/:id/backgrounds`, then re-fetches
- `deleteBackground(channelId, bgId)` — DELETE `/api/channels/:id/backgrounds/:bg_id`, removes from local state and clears `selectedBgId` if it matched
- `selectBackground(id | null)` — sets `selectedBgId`

Uses `useAppStore.getState().goUrl` for the API base URL (no prop drilling).

### BackgroundsGallery.tsx

Client component (`'use client'`). Renders:

- Upload row: optional name text input + "Upload Background" button → hidden `<input type="file" accept="image/*,video/*">`. Shows inline error on failure.
- Grid (`grid-cols-3 sm:grid-cols-4`) of background cards:
  - 16:9 image preview (`aspect-video object-cover`)
  - Filename + star icon for `is_default` backgrounds
  - Delete button (shown on hover via `group-hover:flex`) with per-item loading spinner
  - Selected state: `border-[#00D4FF]` ring + "Selected" badge in top-left corner
- Loading and empty states handled.

### thumbnail/page.tsx (enhanced)

- Replaced `template` state (`branded | centered`) with `strategy` (`branded | centered | template`).
- Added `RadioGroup` + `RadioGroupItem` + `Label` strategy selector at top of form.
- When `strategy === 'template'`:
  - Loads channels on first switch via `useChannelStore.fetchChannels`.
  - Shows `Select` channel picker (shadcn/ui).
  - On channel selection, calls `fetchBackgrounds` and renders `<BackgroundsGallery channelId={...} />`.
  - Warns user when no background is selected; disables Generate button until one is selected.
- `handleGenerate` passes `background_id: selectedBgId` in the POST body when strategy is `template`.
- All existing branded/centered functionality preserved (font color, size, overlay text, output path auto-derivation).
- TypeScript strict — `pnpm tsc --noEmit` exits with zero errors.
