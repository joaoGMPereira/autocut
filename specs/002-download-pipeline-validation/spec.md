# Feature Specification: Download Pipeline — Validation & Hardening

**Feature Branch**: `002-download-pipeline-validation`  
**Created**: 2026-04-12  
**Status**: Draft  
**Input**: User description: "quero validar o que temos do processo de Download no fluxo de pipeline veja como esta o fluxo atual veja o que esta com problema baseado nos logs e veja o que precisamos ajustar para que novas features quando entrem na pipeline nao cause problemas novamente"

## Context

The pipeline executes these steps in order:

```
WAITING_URL → EXECUTING (download) → WAITING_MODE → GENERATING_CLIPS → WAITING_THUMBNAIL_CONFIG → WAITING_REVIEW_METADATA → WAITING_REVIEW_CLIPS → WAITING_UPLOAD_CONFIRM → UPLOADING → DONE
```

The **Download phase** occupies the `EXECUTING` state with `active_phase = 'download'`. The frontend listens to SSE events from the backend and renders a `DownloadInfoCard` showing progress, speed, ETA, video title, and thumbnail.

Code analysis identified the following broken behaviors already present in the codebase:

1. Progress reporting is disconnected — download jumps from 0% to 100% with no intermediate updates.
2. SSE event type is emitted as `"download"` instead of `"phase_progress"`, which the frontend ignores.
3. Speed (KB/s) and ETA (seconds) are never calculated or emitted, so the UI always shows `—`.
4. The downloaded video file path is not persisted to the database — subsequent pipeline phases cannot locate the file.
5. Several state machine, executor, and type files exist as `TODO: Transcribe from Kotlin source` stubs with no implementation.
6. No cancellation: the download goroutine ignores its context; a long-running yt-dlp process cannot be interrupted.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — See live download progress (Priority: P1)

A user submits a YouTube URL and expects to see the download progress bar advance in real time. Currently the bar jumps from 0% to 100% with no movement in between, and speed/ETA always read `—`.

**Why this priority**: This is the primary feedback signal during the download phase. A broken progress bar erodes trust and makes the system appear frozen.

**Independent Test**: Submit a valid YouTube URL → observe the `DownloadInfoCard`. The progress bar must advance steadily, speed must show a non-zero value, and ETA must count down.

**Acceptance Scenarios**:

1. **Given** a user submits a valid YouTube URL, **When** the backend begins downloading, **Then** the frontend receives `phase_progress` events with `percent_done` values between 1 and 99 as the download progresses.
2. **Given** the `phase_progress` event arrives, **When** the frontend processes it, **Then** the progress bar updates and, if `speed_kbs` and `eta_sec` are present, they are displayed in human-readable form.
3. **Given** the download completes, **When** the backend emits the final state change, **Then** the progress bar shows 100% and the UI transitions to the next step.

---

### User Story 2 — Downstream phases receive the downloaded file (Priority: P1)

A user completes the download step and proceeds to transcript / clip generation. Currently the video file path is never stored, so all downstream phases fail silently trying to locate the file.

**Why this priority**: Without this fix no pipeline run can succeed past `WAITING_MODE` — the entire downstream pipeline is blocked.

**Independent Test**: Complete a full download → inspect the pipeline run record — `video_path` must be non-empty. Advance to the next phase — clip generation must start without a "file not found" error.

**Acceptance Scenarios**:

1. **Given** a download completes successfully, **When** the backend transitions to `WAITING_MODE`, **Then** the `video_path` column in the pipeline run record contains the absolute path of the downloaded file.
2. **Given** a subsequent phase starts, **When** it reads the pipeline run record, **Then** it can open the file at `video_path` without error.
3. **Given** a download fails mid-way, **When** the error state is recorded, **Then** `video_path` remains empty and a descriptive error message is stored.

---

### User Story 3 — Cancel an in-progress download (Priority: P2)

A user submits a URL then changes their mind before the download finishes and clicks "Cancel". Currently the backend goroutine ignores the cancellation signal and continues running yt-dlp until it finishes.

**Why this priority**: Without cancellation, users waste bandwidth and time, and the run stays locked in `EXECUTING` until the download completes naturally.

**Independent Test**: Start a download of a long video → click Cancel within 5 seconds → verify the background process terminates and the run transitions to `CANCELLED`.

**Acceptance Scenarios**:

1. **Given** a download is in progress, **When** the user cancels the run, **Then** the background download process is terminated within 3 seconds.
2. **Given** the cancellation succeeds, **When** the backend detects the terminated process, **Then** the run state transitions to `CANCELLED` and a `state_changed` SSE event is emitted.
3. **Given** a cancelled run, **When** the user views the pipeline UI, **Then** the step rail shows the run as cancelled (not stuck in `EXECUTING`).

---

### User Story 4 — Pipeline contract protects future feature additions (Priority: P2)

Future features (highlights, clips, upload) are added to the pipeline. Each new phase needs to know what the download phase guarantees it will produce. Currently there is no contract — developers add code that assumes things the download phase never provides, breaking the pipeline silently.

**Why this priority**: Without a contract, every new phase added to the pipeline risks introducing another silent failure like the `video_path` bug.

**Independent Test**: Review the pipeline contract documentation → confirm that every field listed as "produced by download" is verifiably set in the database after a successful download run.

**Acceptance Scenarios**:

1. **Given** a successful download run, **When** the run transitions to `WAITING_MODE`, **Then** all fields declared in the download contract (`url`, `video_path`, `video_title`, `duration_sec`) are non-empty in the pipeline run record.
2. **Given** a new pipeline phase begins implementation, **When** a developer reads the contract, **Then** they can identify exactly which fields are available and which states are valid entry points.
3. **Given** a phase attempts to start with missing required input fields, **When** the system detects the gap, **Then** it transitions to `ERROR` with a descriptive message instead of panicking or producing silent garbage output.

---

### Edge Cases

- What happens if yt-dlp exhausts all retry strategies (web_client, android, ios, tv_embedded, progressive_720p, best_available)?
- What happens if the video is private, age-restricted, or geo-blocked?
- What happens if metadata extraction succeeds but the actual download fails?
- What happens if the SSE connection drops mid-download and the user reconnects?
- What happens if disk space runs out during download?
- What happens if two pipeline runs try to download to the same output directory concurrently?

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST emit `phase_progress` SSE events with `phase = 'download'` at least every 5 seconds during an active download, with `percent_done` reflecting actual yt-dlp progress.
- **FR-002**: The `phase_progress` event MUST include `speed_kbs` (integer, kilobytes per second) and `eta_sec` (integer, estimated seconds remaining) whenever yt-dlp reports them.
- **FR-003**: The SSE event type for download progress MUST be `"phase_progress"` — not `"download"` or any other value — to match the frontend subscription contract.
- **FR-004**: On successful download completion, the system MUST persist the absolute file path of the downloaded video to `video_path` on the pipeline run record before transitioning to `WAITING_MODE`.
- **FR-005**: The download goroutine MUST honour context cancellation: when the run is cancelled, the yt-dlp subprocess MUST be terminated and the run state MUST transition to `CANCELLED`.
- **FR-006**: The system MUST NOT transition to `WAITING_MODE` if `video_path` is empty after a completed download — it MUST instead transition to `ERROR` with a descriptive message.
- **FR-007**: The pipeline MUST define a **phase output contract** — a structured declaration of which fields each phase writes to the run record and which fields it requires from previous phases.
- **FR-008**: Each phase entry point MUST validate its required input fields from the contract before executing, and transition to `ERROR` with a descriptive message if any required field is missing.
- **FR-009**: All `TODO: Transcribe from Kotlin source` stub files (`state_machine.go`, `types.go`, `executor.go`, `sse.go`) MUST either be fully implemented or removed; no empty stubs that are currently imported may remain.
- **FR-010**: Platform detection MUST validate the URL before attempting download, returning a user-readable error for unsupported or malformed URLs instead of propagating a nil pointer panic.

### Key Entities

- **PipelineRun**: Single execution of the pipeline. Key fields: `id`, `state`, `active_phase`, `url`, `video_path`, `video_title`, `duration_sec`, `error`, `started_at`, `finished_at`.
- **PhaseContract**: Declared set of fields that a phase produces (outputs) and consumes (inputs). Defined in code, not just documentation.
- **SSEEvent**: Server-sent event emitted to the frontend. Types: `state_changed`, `phase_progress`, `video_info`, `error`. Each type has a fixed payload schema.
- **DownloadResult**: Output of a successful yt-dlp invocation. Fields: `FilePath`, `Title`, `ThumbnailURL`, `DurationSec`.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: During a standard download (video ≤ 30 min), the progress bar advances at least 5 times between 0% and 100% — no freeze at 0%.
- **SC-002**: After every successful download, `video_path` in the run record is non-empty and points to a readable file.
- **SC-003**: A pipeline run cancelled during download terminates the background process within 3 seconds of the cancellation request.
- **SC-004**: Speed and ETA fields are populated in `phase_progress` events for at least 80% of updates (excluding the initial 0% and the final 100%).
- **SC-005**: Zero pipeline runs transition to `WAITING_MODE` with an empty `video_path` — all such cases route to `ERROR` instead.
- **SC-006**: Any new phase that fails its input contract validation produces a descriptive error message visible in the pipeline UI within 2 seconds.
- **SC-007**: No `TODO: Transcribe from Kotlin source` stub files remain in the pipeline server code tree.

---

## Assumptions

- yt-dlp is installed and accessible on the server PATH; its progress output format (JSON lines) does not change between currently deployed versions.
- The frontend SSE subscription logic is correct and only needs the backend to emit properly typed events — no frontend changes are needed to display progress once the backend emits the right event types.
- The database schema already contains `video_path` and `duration_sec` columns on `pipeline_runs` — no schema migration is needed for these fields.
- Disk space management (cleanup of downloaded files after pipeline completion) is out of scope for this feature.
- Multi-platform support beyond YouTube and Twitch is out of scope — focus is on hardening what already exists.
- Cancellation will use Go context propagation passed into the download goroutine; no new external cancel API endpoint is needed.
