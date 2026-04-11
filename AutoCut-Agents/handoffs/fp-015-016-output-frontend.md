---
feature: FP-015+016-YouTubeFrontend
stage: output
scope: frontend
status: approved
files_created:
  - AutoCut/apps/web/src/store/massUpdateStore.ts
  - AutoCut/apps/web/src/components/channels/CommentSyncSheet.tsx
files_modified:
  - AutoCut/apps/web/src/app/mass-update/page.tsx
  - AutoCut/apps/web/src/app/channels/page.tsx
---

## Summary

Implemented the full frontend for FP-015 (Mass Update) and FP-016 (Comment Sync UI).

### massUpdateStore.ts

Zustand store exposing:
- `fetchVideos(goUrl, page?)` — GET /api/videos/remote with channel_id + filter params
- `runDryRun(goUrl)` — POST /api/videos/mass-update?dry_run=true → stores preview
- `runUpdate(goUrl)` — POST /api/videos/mass-update → stores job_id
- `subscribeProgress(goUrl)` — opens EventSource on GET /api/videos/mass-update/:id/stream, returns unsubscribe fn
- Full filter (date_from, date_to, category_id) and update (title_template, description_append, tags_add/remove, hashtags, category_id) state

### mass-update/page.tsx

Replaced placeholder with real interactive page:
- Channel selector — loads remote videos on change
- FilterBar — date range + category, apply button
- VideoList — paginated with prev/next, shows video count
- UpdateForm — title template (with placeholder hints), description append, tags add/remove, hashtags, category
- ActionButtons — "Preview (Dry Run)" + "Apply to All Videos" (confirm dialog before applying)
- DryRunPreview — table with youtube_id / current_title / new_title, affected count badge
- ProgressPanel — subscribes to SSE when job starts, progress bar, done summary
- ErrorBanner — shows error state

### CommentSyncSheet.tsx

Sheet component (no separate store):
- Loads comments on open via GET /api/channels/:id/comments
- "Sync Comments" button — POST /api/channels/:id/comments/sync → SSE progress bar
- After sync completes, reloads comment list
- Search input (debounced-by-keystroke)
- Comment list: author, truncated text (200 chars), likes, date
- Prev/Next pagination

### channels/page.tsx

Added "Comments" button (MessageSquare icon) next to "Configure" for each channel row. Opens CommentSyncSheet via `commentsChannel` state.
