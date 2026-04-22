# Design: Review Metadata Screen + Queue Enrichment

**Date:** 2026-04-21
**Branch:** 114-review-screens-refactor

## Overview

Three focused improvements to the pipeline review + upload queue flow:

1. **StepReviewMetadata** — add thumbnail type badge per clip card
2. **Queue page** — rich cards (thumbnail + channel name) + channel filter pills
3. **Upload queue backend** — fix missing thumbnail path correlation, write metadata.json to disk, hardcode made_for_kids=false

---

## Part 1: StepReviewMetadata — Thumbnail Type Badge

### Scope
Frontend only. No migration, no backend changes.

### What changes
In `apps/web/src/components/pipeline/steps/StepReviewMetadata.tsx`, the header row of each clip card gains a small badge showing the thumbnail style:

- `clip.thumbnail_style === 'landscape'` → badge "Landscape" (amber color)
- anything else (including `'default'`, `'shorts'`) → badge "Shorts" (violet color)

Position: inline in the card header row, between the existing "Clip N" badge and the duration span.

### Data source
`clip.thumbnail_style` is already included in `PipelineClip` loaded via `loadClips` — no new fields needed.

### Component change
```
// Header row (existing)
<Badge>Clip {idx+1}</Badge>
<span>{duration}</span>

// After change
<Badge>Clip {idx+1}</Badge>
<ThumbnailTypeBadge style={clip.thumbnail_style} />   ← new
<span>{duration}</span>
```

`ThumbnailTypeBadge` is a small inline component (defined in the same file, ~8 lines).

---

## Part 2: Queue Page — Rich Cards + Channel Filter

### Scope
Frontend: `QueuePage`, `QueueItemCard`, `queueStore`.
Backend: `ListQueueWithTitle` SQL query, `QueueItemRow` struct, one new endpoint.

### 2a — Backend changes

**`ListQueueWithTitle` SQL** — join `pipeline_clips.thumbnail_path`:
```sql
SELECT u.id, u.clip_id, u.channel_id, u.status,
       COALESCE(u.local_video_path, '') as video_path,
       u.queue_order, u.publish_at,
       u.youtube_id, u.youtube_url, u.error, u.created_at,
       COALESCE(c.title, '') as title,
       COALESCE(c.thumbnail_path, '') as thumbnail_path,   -- NEW
       CASE WHEN COALESCE(c.duration_sec, 999) <= 60 THEN 'short' ELSE 'long_form' END as video_type
FROM uploads u
LEFT JOIN pipeline_clips c ON c.id = u.clip_id
WHERE u.status IN ('queued', 'running', 'failed', 'error', 'uploaded')
ORDER BY u.queue_order ASC, u.created_at ASC
```

**`QueueItemRow`** gains field `ThumbnailPath string`.

**New endpoint** `GET /api/queue/{id}/thumbnail`:
- Loads upload record by id
- Reads `local_thumbnail_path` from DB (after Part 3 fix, this is populated)
- Serves the file with `Content-Type: image/jpeg`
- Falls back to `pipeline_clips.thumbnail_path` if `local_thumbnail_path` is empty
- Returns 404 if neither exists

**Queue list endpoint** includes `thumbnail_path` field in the JSON response so the frontend can construct a URL:
`{goUrl}/api/queue/{id}/thumbnail`

### 2b — Frontend changes

**`QueueItem` type** gains:
```ts
thumbnail_path?: string;
channel_name?: string;  // resolved client-side, not from API
```

**`queueStore`**: `QueueItem` updated to include `thumbnail_path` from API response.

**`QueuePage`**:
- Imports `useChannelStore`, calls `fetchChannels` on mount if needed
- Adds `selectedChannelId: number | null` state (null = all channels)
- Derives `channelOptions` from `items` — unique `channel_id` values with names from channelStore
- Renders filter pills above the list: "Todos" + one pill per channel that has items
- Filters: `visibleItems = items.filter(i => !selectedChannelId || i.channel_id === selectedChannelId)`
- `autoSchedule` uses `selectedChannelId ?? firstQueued.channel_id`

**`QueueItemCard`** props gain `channelName?: string`. Card layout becomes:
```
[thumbnail 80×60] | [title + status badge + channel badge]
                   | [publish_at / error / youtube link]
                   | [action buttons]
```
Thumbnail is 80px wide, aspect-video, `object-cover`, rounded corner. Falls back to a grey placeholder if no thumbnail. Channel shown as a small dim badge below the title.

---

## Part 3: Upload Queue — Correlation Fix + metadata.json + made_for_kids

### Scope
Backend only. No frontend changes, no migration.

### 3a — Bug fix: save thumbnail path in uploads table

**Problem:** `PostUploadConfirm` in `service.go` discards `queuedThumb`:
```go
_ = queuedThumb // stored in metaJSON; thumbnail_path not a separate uploads column
```
But `uploads.local_thumbnail_path` column exists and is never populated.

**Fix:**
- `UploadRepo.CreateQueued` gains `thumbnailPath string` parameter
- INSERT sets `local_thumbnail_path = ?`
- `PostUploadConfirm` passes `queuedThumb` to `CreateQueued`

### 3b — Write metadata.json to upload_queue folder

**`QueueStorage.SaveToQueue`** signature gains `metadata VideoMetadata` parameter.

After copying video + thumbnail, writes `metadata.json` to the upload dir:
```json
{
  "channel_id": 3,
  "title": "...",
  "description": "...",
  "tags": ["tag1", "tag2"],
  "category_id": "24",
  "privacy": "private",
  "made_for_kids": false,
  "language": "pt-BR"
}
```

File is human-readable and useful for debugging. DB (`metadata_json` column) remains the source of truth — the file is derived from it.

### 3c — made_for_kids hardcoded false

**`BuildMetadata`** (`uploader/metadata.go`) sets `MadeForKids: false` (already the default since `channel_config.made_for_kids` defaults to 0).

**`service.go` line ~115** override removed:
```go
// REMOVE: metadata.MadeForKids = cfg.MadeForKids
```

This ensures made_for_kids is never accidentally set to true from channel config data.

---

## Data flow after all changes

```
Pipeline clip generated
    ↓
StepReviewMetadata — user edits title/desc/tags, sees thumbnail type badge
    ↓
StepUpload confirmed (queue mode)
    ↓
PostUploadConfirm:
  QueueStorage.SaveToQueue(videoPath, thumbnailPath, metadata)
    → upload_queue/{ts}_{rnd}/video.mp4
    → upload_queue/{ts}_{rnd}/thumbnail.jpg
    → upload_queue/{ts}_{rnd}/metadata.json
  UploadRepo.CreateQueued(channelID, clipID, videoPath, thumbnailPath, metaJSON, configJSON, 0)
    → uploads.local_video_path    = upload_queue/.../video.mp4
    → uploads.local_thumbnail_path = upload_queue/.../thumbnail.jpg
    → uploads.metadata_json       = { title, desc, tags, category, privacy, made_for_kids: false }
    ↓
Queue page — shows thumbnail + title + channel name, filterable by channel
```

---

## Files affected

| File | Change |
|------|--------|
| `apps/web/src/components/pipeline/steps/StepReviewMetadata.tsx` | Add `ThumbnailTypeBadge` component + usage |
| `apps/web/src/app/queue/page.tsx` | Channel filter pills, channelStore, filtered items |
| `apps/web/src/components/queue/QueueItemCard.tsx` | Thumbnail image + channel badge |
| `apps/web/src/store/queueStore.ts` | `QueueItem` type gains `thumbnail_path` |
| `server/internal/database/upload_repo.go` | `CreateQueued` + `ListQueueWithTitle` + `QueueItemRow` |
| `server/internal/uploader/storage.go` | `SaveToQueue` gains `metadata` param, writes metadata.json |
| `server/internal/uploader/metadata.go` | `MadeForKids: false` hardcoded |
| `server/internal/pipeline/service.go` | Pass `queuedThumb` to `CreateQueued`, remove MadeForKids override |
| `server/internal/api/handlers/` (queue handler) | New thumbnail endpoint + updated list response |

## No schema migration needed

All changes use existing columns (`local_thumbnail_path` already exists in v1 schema + rebuilt in v19). No new columns, no new tables.
