# AutoCut Constitution

## Mission

AutoCut v2 is a complete rewrite of Youtube v1 (Kotlin + Compose Desktop) into
Electron + Go 1.23 + Next.js 16. The mission is full feature parity with Youtube v1,
with improved performance and better cross-platform configuration.

**No feature is considered DONE until it passes the transcription process below.**
Existing code in AutoCut v2 is treated as a draft. Features are rebuilt feature by
feature, class by class, from the Kotlin source.

## Transcription Process (mandatory per feature)

```
1. ANALYZE    → Open the Kotlin reference class(es) in /Youtube
2. MAP        → Understand responsibilities, delegates, dependencies, edge cases
3. TRANSCRIBE → Implement in Go (backend) or TypeScript (frontend/electron)
4. VALIDATE   → Run the feature's validation script (exit 0 required)
5. FRONTEND   → Implement plan-frontend.md + run validate-{n}-fe.sh (exit 0 required)
               [Skip if FE = N/A]
6. MARK DONE  → Update BE Status + FE column in Feature Parity Tracker below + commit
```

Rules:
- The agent opens the Kotlin source before writing any line of Go/TypeScript
- Implementation must be functionally equivalent — not a copy, not an invention
- Each feature has a dedicated validation script in `scripts/validate/`
- Status never moves to DONE without the script passing
- FE work is gated: Phase 7 of tasks.md unlocks only after validate-{n}.sh exits 0

## Architecture

### Layer Responsibilities

| Layer | Path | Rule |
|---|---|---|
| Business Logic | `server/internal/` | All logic lives here — never in handlers |
| HTTP Handlers | `server/internal/api/handlers/` | Routing + request parsing only |
| Database | `server/internal/database/` | Repository pattern — one repo per entity |
| UI | `apps/web/src/` | UI only — no filesystem or business logic |
| Desktop | `apps/desktop/src/` | Native IPC only — no business logic |
| Shared | `packages/shared/` | Types + hooks only — no side effects |

### Tech Stack (immutable)

- **Backend:** Go 1.23, slog, SQLite (modernc.org/sqlite, CGO_ENABLED=0), WAL mode
- **Frontend:** Next.js 16, React, Tailwind CSS, TypeScript 5.8
- **Desktop:** Electron, TypeScript, IPC for native APIs only
- **External tools:** FFmpeg, yt-dlp, TwitchDownloaderCLI, Whisper.cpp, Ollama

### Key Constraints

- CGO_ENABLED=0 always — pure Go sqlite driver required for cross-compilation
- WAL mode required on all SQLite connections — concurrent write safety
- No business logic in HTTP handlers — untestable
- IPC only for native capabilities — no app logic in Electron main process

### Progress Reporting (SSE)

All long-running backend operations (download, transcription, optimization, upload, etc.)
**must** report progress through a single generic SSE endpoint. No operation-specific
progress channels — one mechanism, many producers.

**Contract**:
- Endpoint: `GET /api/events?job_id=<id>` — streams `text/event-stream`
- Event envelope (JSON):
  ```json
  { "job_id": "…", "stage": "download|transcribe|optimize|upload|…",
    "pct": 0–100, "message": "human-readable status", "error": null | "…" }
  ```
- A terminal event (`pct: 100` or non-null `error`) closes the stream
- Business logic emits progress via an injected `ProgressReporter` interface — never directly to HTTP
- Handlers wire the `ProgressReporter` to the SSE connection; business logic stays unaware of HTTP

**Rules for agents**:
- Every feature that takes > 1 s must emit at least START (0%), key milestones, and DONE (100%)
- `ProgressReporter` is an interface in `server/internal/progress/` — inject it, do not import `net/http` from business logic
- UI subscribes to `/api/events?job_id=<id>` — no polling

## Agent Pipeline

```
@act-orchestrator
  → classifies scope: infra | backend | frontend | full
  → activates teams:
      [backend]  @act-dev-backend  → @act-qa
      [frontend] @act-dev-frontend → @act-qa
      [infra]    @act-dev-infra    → @act-qa
  → after 3 QA rejections → escalate to human
```

Handoff rules:
- YAML handoff required — schema: `skills/core/handoff.md`
- `pending_decisions` must be empty before passing to next agent
- Agents do not reconstruct context from memory — context comes from handoff
- Scope vocabulary: `infra | backend | frontend | full`

## Feature Parity Tracker

Status values: `TODO` | `IN PROGRESS` | `NEEDS VALIDATION` | `DONE`

**DONE** = validation script exits 0 + PR merged into main.

### Download

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 1 | Download YouTube | `YouTubeDownloader.kt` + `VideoDownloadDelegate.kt` | Go | NEEDS VALIDATION | NEEDS VALIDATION |
| 2 | Download Twitch VOD | `TwitchDownloader.kt` | Go | TODO | N/A |
| 3 | Download Twitch Clips | `TwitchClipDownloader.kt` | Go | TODO | N/A |
| 4 | Download Twitch Chat | `TwitchChatDownloader.kt` | Go | TODO | N/A |
| 5 | Thumbnail Download | `ThumbnailDownloadDelegate.kt` | Go | NEEDS VALIDATION (implemented in FP-001 as `thumbnail.go`; covered by `validate-01.sh`) | N/A |
| 6 | Snippet/Metadata Download | `SnippetDownloadDelegate.kt` | Go | TODO | N/A |

### Transcription

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 7 | Whisper Transcription | `TranscriptGenerator.kt` | Go | TODO | N/A |
| 8 | Audio Extraction | `TranscriptAudioDelegate.kt` | Go | TODO | N/A |
| 9 | Long Video Chunking | `TranscriptChunkDelegate.kt` | Go | TODO | N/A |
| 10 | Transcript Cache | `TranscriptCacheDelegate.kt` | Go | TODO | N/A |
| 11 | JSON Parsing + Post-processing | `TranscriptJsonParser.kt` | Go | TODO | N/A |
| 12 | Subtitle Generation | `SubtitleGenerator.kt` | Go | TODO | N/A |

### AI Analysis

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 13 | Topic Classification | `TopicClassificationDelegate.kt` | Go | TODO | N/A |
| 14 | Explicit Transition Detection | `TopicExplicitTransitionDelegate.kt` | Go | TODO | N/A |
| 15 | Semantic Transition Detection | `TopicSemanticTransitionDelegate.kt` | Go | TODO | N/A |
| 16 | Music/Noise Filter | `TopicMusicFilterDelegate.kt` | Go | TODO | N/A |
| 17 | Speech Pattern Detection | `SpeechPatternDetector.kt` + `PatternDetectionDelegate.kt` | Go | TODO | N/A |
| 18 | Pattern Cache | `PatternCacheDelegate.kt` | Go | TODO | N/A |

### Highlight Detection

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 19 | Highlight Detector | `HighlightDetector.kt` | Go | TODO | N/A |
| 20 | Audio/Silence Analysis | `AudioAnalyzer.kt` | Go | TODO | N/A |
| 21 | Scene Analysis | `SceneAnalyzer.kt` | Go | TODO | N/A |
| 22 | Detection Strategy Selection | `DetectionStrategy.kt` | Go | TODO | N/A |

### Clip Generation

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 23 | Clip Cutting | `ClipProcessorService.kt` | Go | TODO | N/A |
| 24 | Clip Review | `ClipReviewService.kt` | Go | TODO | N/A |
| 25 | Text Overlay | `TextOverlayProcessor.kt` | Go | TODO | N/A |
| 26 | Video Overlay | `VideoOverlayProcessor.kt` | Go | TODO | N/A |
| 27 | Preview Generation | `PreviewProcessor.kt` | Go | TODO | N/A |
| 28 | Uniqueness Detection | `VideoUniquenessProcessor.kt` | Go | TODO | N/A |

### Shorts

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 29 | Shorts Generator | `ShortsGenerator.kt` | Go | TODO | N/A |
| 30 | Segment Detection | `ShortsSegmentDetector.kt` | Go | TODO | N/A |
| 31 | Vertical Video (9:16) | `VerticalVideoProcessor.kt` | Go | TODO | N/A |
| 32 | Shorts Cache | `ShortsCacheDelegate.kt` | Go | TODO | N/A |
| 33 | Parallel Generation (5 jobs) | `ShortsParallelGenerationDelegate.kt` | Go | TODO | N/A |
| 34 | Shorts Subtitles | `ShortsSubtitleDelegate.kt` | Go | TODO | N/A |
| 35 | Title Generation | `TitleGenerator.kt` | Go | TODO | N/A |

### Thumbnail

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 36 | Strategy — Centered | `CenteredThumbnailStrategy.kt` | Go | TODO | N/A |
| 37 | Strategy — Branded | `BrandedStrategy.kt` | Go | TODO | N/A |
| 38 | Strategy — BestFrame | `BestFrameStrategy.kt` | Go | TODO | N/A |
| 39 | Strategy — LongForm | `LongFormStrategy.kt` | Go | TODO | N/A |
| 40 | Strategy — Shorts | `ShortsThumbnailStrategy.kt` | Go | TODO | N/A |
| 41 | Strategy — Simple Overlay | `SimpleOverlayStrategy.kt` | Go | TODO | N/A |
| 42 | Strategy — Template | `TemplateStrategy.kt` | Go | TODO | N/A |
| 43 | Font Selection | `ThumbnailFontDelegate.kt` | Go | TODO | N/A |
| 44 | Color Customization | `ThumbnailColorsDelegate.kt` | Go | TODO | N/A |
| 45 | Presets (Shorts/LongForm/Live/React) | `ThumbnailPresets.kt` | Go | TODO | N/A |
| 46 | FFmpeg Text Renderer | `FFmpegTextRenderer.kt` + `FFmpegTextOverlayDelegate.kt` | Go | TODO | N/A |
| 47 | System Font Detection | `SystemFontDetector.kt` | Go | TODO | N/A |

### Upload & Auth

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 48 | YouTube Upload | `YouTubeUploader.kt` | Go | TODO | N/A |
| 49 | OAuth 2.0 | `OAuthAuthenticator.kt` + `AuthenticationDelegate.kt` | Go | TODO | N/A |
| 50 | Client Secret Management | `ClientSecretService.kt` | Go | TODO | N/A |
| 51 | Thumbnail Upload | `ThumbnailUploadDelegate.kt` | Go | TODO | N/A |
| 52 | Upload Progress | `UploadProgressDelegate.kt` | Go | TODO | N/A |
| 53 | Upload Validation | `UploadValidationDelegate.kt` | Go | TODO | N/A |
| 54 | Playlist Management | `PlaylistManagementDelegate.kt` | Go | TODO | N/A |
| 55 | Post-upload Metadata Update | `VideoManagementDelegate.kt` | Go | TODO | N/A |
| 56 | Comment Management | `CommentManagementDelegate.kt` | Go | TODO | N/A |
| 57 | Quota Tracking | `QuotaTrackingDelegate.kt` | Go | TODO | N/A |

### Scheduling & Queue

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 58 | Scheduling Service | `SchedulingService.kt` + `SchedulingDelegate.kt` | Go | TODO | N/A |
| 59 | Upload Queue | `UploadQueueService.kt` | Go | TODO | N/A |
| 60 | Privacy Status (public/private/unlisted) | `PrivacyStatus.kt` | Go | TODO | N/A |

### Channel Metadata

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 61 | Channel Management | `ChannelsViewModel.kt` + `ChannelImportService.kt` | Go | TODO | N/A |
| 62 | Channel Preferences | `ChannelPreferencesService.kt` | Go | TODO | N/A |
| 63 | Channel Logo | `ChannelLogoService.kt` | Go | TODO | N/A |
| 64 | Metadata Service | `ChannelMetadataService.kt` | Go | TODO | N/A |
| 65 | Browser Cookie Extraction | `BrowserCookieDelegate.kt` | Go | TODO | N/A |
| 66 | Metadata Cache | `MetadataCacheDelegate.kt` | Go | TODO | N/A |
| 67 | URL Normalization | `URLNormalizerDelegate.kt` | Go | TODO | N/A |
| 68 | Parallel Metadata Processing | `ParallelProcessorDelegate.kt` | Go | TODO | N/A |
| 69 | AI Prompt Generation | `PromptGeneratorDelegate.kt` | Go | TODO | N/A |
| 70 | Metadata History | `ChannelMetadataHistoryService.kt` | Go | TODO | N/A |

### Optimizer

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 71 | Audio Silence Detector | `AudioSilenceDetector.kt` | Go | TODO | N/A |
| 72 | Composite Silence Detector | `CompositeSilenceDetector.kt` | Go | TODO | N/A |
| 73 | Transcript Pause Detector | `TranscriptPauseDetector.kt` | Go | TODO | N/A |
| 74 | Speed Zones | `SpeedChainBuilder.kt` + `SpeedZoneDelegate.kt` | Go | TODO | N/A |
| 75 | Music Overlay | `MusicLibraryService.kt` | Go | TODO | N/A |
| 76 | Transition Filter | `TransitionFilterBuilder.kt` + `ComplexFilterBuilder.kt` | Go | TODO | N/A |
| 77 | Chunked Optimization | `ChunkedVideoOptimizer.kt` | Go | TODO | N/A |
| 78 | Optimization Cache | `OptimizationCacheDelegate.kt` | Go | TODO | N/A |

### Post-Optimization

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 79 | Video Splitting | `PostOptSplitDelegate.kt` | Go | TODO | N/A |
| 80 | Intro/Outro Insertion | `PostOptPatternDelegate.kt` | Go | TODO | N/A |
| 81 | Per-part Metadata | `PostOptMetadataDelegate.kt` | TS | TODO | N/A |
| 82 | Auto-thumbnail per part | `PostOptThumbnailDelegate.kt` | Go | TODO | N/A |
| 83 | Batch Upload Parts | `PostOptUploadDelegate.kt` | Go | TODO | N/A |

### Settings & Configuration

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 84 | Tool Path Configuration | `ToolsConfigDelegate.kt` | Go | TODO | N/A |
| 85 | Installation Service | `InstallationService.kt` | Go | TODO | N/A |
| 86 | Tool Path Auto-detection | `ToolPathDetectionDelegate.kt` | Go | TODO | N/A |
| 87 | Channel Styling | `ChannelStylingDelegate.kt` | Go+TS | TODO | N/A |
| 88 | Background Image Management | `ChannelBackgroundsDelegate.kt` | Go | TODO | N/A |
| 89 | Default Tags / Categories | `ChannelContentDelegate.kt` | Go | TODO | N/A |
| 90 | App Settings Persistence | `AppSettingsService.kt` | Go | TODO | N/A |

### Additional Features

| # | Feature | Kotlin Reference | Target | BE Status | FE |
|---|---|---|---|---|---|
| 91 | Mass Update | `MassUpdateViewModel.kt` | TS | TODO | N/A |
| 92 | Update Checker | `UpdateCheckerService.kt` | Electron | TODO | N/A |
| 93 | Background Service / Queue Worker | `BackgroundService.kt` | Go | TODO | N/A |
| 94 | Queue View + Reorder + Cancel + Retry | `QueueViewModel.kt` | TS | TODO | N/A |
| 95 | Pipeline Workflow | All `*WorkflowDelegate.kt` steps | Go | TODO | N/A |
| 96 | SSE Progress Reporting | *(new in v2 — no Kotlin equivalent)* | Go | TODO | N/A |

## Quality Gates

Every feature must pass before status → DONE:

1. Validation script in `scripts/validate/validate-{feature}.sh` exits 0
2. `go test ./...` passes (backend features)
3. `pnpm test` passes (frontend features)
4. No business logic in handlers (backend)
5. No filesystem calls in UI components (frontend)
6. Structured logging at every layer

## Governance

1. This constitution supersedes `AutoCut/CLAUDE.md` and all other instruction files
2. The Feature Parity Tracker is updated in the same commit as the feature implementation
3. `scripts/validate/validate-parity.sh` aggregates all feature scripts — must pass before PR merge
4. Amendments require human approval + version bump
5. Agents consult this file before starting any feature work

**Version**: 1.1.0 | **Ratified**: 2026-04-10 | **Last Amended**: 2026-04-10
