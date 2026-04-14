# Phase Contract: Download

**Phase ID**: `download`  
**Entry state**: `EXECUTING` (with `active_phase = 'download'`)  
**Exit state (success)**: `WAITING_MODE`  
**Exit state (failure)**: `ERROR`  
**Exit state (cancel)**: `CANCELLED`

---

## Inputs (required before phase can start)

The download phase validates these fields on the `pipeline_runs` record **before** starting any external process. If any are missing, the phase transitions to `ERROR` immediately.

| DB Column | Type | Constraint | Provided by |
|-----------|------|------------|-------------|
| `url` | TEXT | non-empty; valid http/https URL; supported platform | WAITING_URL advance call |

---

## Outputs (written before transitioning to WAITING_MODE)

All three fields are written atomically via `SetDownloadResult()` in a single UPDATE. If the update fails or any field is empty/zero after a successful download, the phase transitions to `ERROR` instead of `WAITING_MODE`.

| DB Column | Type | Constraint | Description |
|-----------|------|------------|-------------|
| `video_path` | TEXT | non-empty absolute path; file must exist | Absolute path to the downloaded `.mp4` file on disk |
| `video_title` | TEXT | non-empty | Human-readable title from yt-dlp metadata |
| `duration_sec` | INTEGER | > 0 | Video duration in whole seconds |

---

## SSE Events emitted during this phase

Events are published to `hub.SSEHub` under key `fmt.Sprintf("%d", runID)`.

| Event type | When | Payload |
|------------|------|---------|
| `state_changed` | Phase starts | `{run_id, state: "EXECUTING"}` |
| `phase_progress` | Phase start (0%) | `{run_id, phase:"download", percent_done:0}` |
| `video_info` | After metadata extraction | `{run_id, title, thumbnail_url, duration_sec}` |
| `phase_progress` | Each yt-dlp progress line (1–99%) | `{run_id, phase:"download", percent_done, speed_kbs?, eta_sec?}` |
| `phase_progress` | Phase complete (100%) | `{run_id, phase:"download", percent_done:100}` |
| `state_changed` | Phase complete | `{run_id, state:"WAITING_MODE"}` |
| `error` | Any failure | `{run_id, state:"ERROR", message}` |
| `cancelled` | User cancellation | `{run_id, state:"CANCELLED"}` |

---

## Guarantees

1. **video_path is readable**: The file at `video_path` exists on disk and is non-empty at the moment `WAITING_MODE` is entered.
2. **video_title is non-empty**: Set from yt-dlp metadata; never empty if phase succeeded.
3. **duration_sec > 0**: Set from yt-dlp metadata; always positive if phase succeeded.
4. **Atomicity**: All three output fields are written in a single SQL UPDATE (`SetDownloadResult`) before `Finish()` transitions the state.
5. **No partial transitions**: `Finish("WAITING_MODE")` is only called after `SetDownloadResult` succeeds. If `SetDownloadResult` fails, `Finish("ERROR")` is called instead.

---

## What downstream phases can depend on

Any phase that follows `WAITING_MODE` may read and rely on:
- `pipeline_runs.video_path` → absolute path to the source video file
- `pipeline_runs.video_title` → title for metadata, clip naming
- `pipeline_runs.duration_sec` → total duration for highlight detection, progress estimation

Any phase that reads these fields MUST call `ValidatePhaseInputs(run, PhaseTranscribe)` (or equivalent) at entry, transitioning to `ERROR` with a descriptive message if any are missing.

---

## What this phase does NOT guarantee

- Thumbnail stored on disk (delivered via SSE `video_info` only; not persisted to DB)
- File format beyond `.mp4` (yt-dlp merge step produces mp4 but is not guaranteed for all fallback strategies)
- File quality (quality is best-effort via strategy cascade: web → android → ios → tv_embedded → progressive_720p → best_available)
