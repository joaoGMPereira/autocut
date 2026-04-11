# Design: Quick Repeat Pipeline with Local Video Reuse

**Date:** 2026-04-02
**Status:** Approved

---

## Context

When a user wants to re-process a video they already ran through the pipeline (e.g., different mode, different config), they currently have to start from scratch: go through the canal step, paste the URL again, wait for the video to download again. The video is already on disk at `~/.autocut/pipeline/{sourceRunId}/`, wasted.

This feature adds a **"Refazer"** (Repeat) button to every recent run card. Clicking it creates a new pipeline run pre-populated from the source run, opens the wizard directly at the Mode step, and reuses the already-downloaded local video file — skipping the download step entirely. If the local file no longer exists, it falls back to a normal download.

---

## Architecture

### Frontend Changes

**`RecentRunsSection.tsx`** — Add "Refazer" button to each run card alongside the existing "Retomar" button.

**`wizardStore.ts`** — Add two new fields to wizard state:
- `sourceRunId: number | null` — ID of the run being repeated
- `localFilePath: string | null` — resolved path of the existing video file

**`PipelineWizardLayout.tsx`** — Add a `startFromRepeat(run: PipelineRun)` function that:
1. Sets `pendingUrl` from `run.url`
2. Sets `pendingChannelId` from `run.channelId`
3. Sets `sourceRunId` and `localFilePath` (resolved via `run.stepOutputs.download.video_path`)
4. Jumps wizard directly to the Mode step (step index 2)

**`UrlStepPanel.tsx`** — No changes needed (URL step is skipped in this flow).

**`createRun` API call** — Pass `source_run_id` in the POST body when creating the new run.

### Backend Changes

**`POST /api/pipeline/runs`** — Accept optional `source_run_id int` in request body. Store it in the new `pipeline_runs.source_run_id` column.

**`pipeline/service.go` — `executeDownloadStep()`** — Before calling yt-dlp:
1. If `source_run_id` is set, resolve `~/.autocut/pipeline/{sourceRunId}/` and find the video file
2. If file exists: set `StepOutput.VideoPath` to the resolved local file path and mark step as done (no file copy — path reference only)
3. If file does not exist: proceed with normal download using `run.URL`

**Database migration** — Add `source_run_id INTEGER REFERENCES pipeline_runs(id)` column to `pipeline_runs`.

---

## Data Flow

```
User clicks "Refazer" on recent run card
    ↓
startFromRepeat(run) in PipelineWizardLayout
    ├─ wizardStore: set pendingUrl, pendingChannelId, sourceRunId, localFilePath
    └─ jump to Mode step (skip Canal + URL steps)
    ↓
User configures mode → clicks "Executar"
    ↓
POST /api/pipeline/runs { url, channel_id, source_run_id }
    ↓
Backend ExecuteRun():
    ├─ executeDownloadStep():
    │   ├─ if source_run_id set AND local file exists → use file, skip download
    │   └─ else → normal yt-dlp download
    └─ remaining steps proceed normally
```

---

## UI Details

- **Button label:** "Refazer" (small, secondary style, next to "Retomar")
- **Tooltip:** "Nova pipeline com o mesmo vídeo"
- Run cards that have `status: pending` or `status: running` — "Refazer" is disabled (no completed download yet)
- Run cards where `stepOutputs.download` is absent — "Refazer" falls back to full download (button still shown, no special treatment needed since backend handles it)

---

## Error Handling

- Local file missing at execution time → silent fallback to download (no error shown)
- Source run deleted from DB before new run executes → `source_run_id` lookup returns nothing → download proceeds normally
- No changes needed to existing SSE progress reporting

---

## Database

```sql
ALTER TABLE pipeline_runs
  ADD COLUMN source_run_id INTEGER REFERENCES pipeline_runs(id);
```

---

## Testing

1. Run a pipeline to completion → video exists at `~/.autocut/pipeline/{id}/`
2. Click "Refazer" on that card → wizard opens at Mode step with URL pre-filled
3. Execute → verify download step is skipped in logs/progress
4. Delete the video file manually → click "Refazer" again → verify download happens normally
5. Verify "Refazer" appears disabled on a run with status `pending`
