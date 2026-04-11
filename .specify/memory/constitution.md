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
5. MARK DONE  → Update status in Feature Parity Tracker below + commit
```

Rules:
- The agent opens the Kotlin source before writing any line of Go/TypeScript
- Implementation must be functionally equivalent — not a copy, not an invention
- Each feature has a dedicated validation script in `scripts/validate/`
- Status never moves to DONE without the script passing

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

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 1 | Download YouTube | `YouTubeDownloader.kt` + `VideoDownloadDelegate.kt` | Go | NEEDS VALIDATION |
| 2 | Download Twitch VOD | `TwitchDownloader.kt` | Go | TODO |
| 3 | Download Twitch Clips | `TwitchClipDownloader.kt` | Go | TODO |
| 4 | Download Twitch Chat | `TwitchChatDownloader.kt` | Go | TODO |
| 5 | Thumbnail Download | `ThumbnailDownloadDelegate.kt` | Go | TODO |
| 6 | Snippet/Metadata Download | `SnippetDownloadDelegate.kt` | Go | TODO |

### Transcription

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 7 | Whisper Transcription | `TranscriptGenerator.kt` | Go | TODO |
| 8 | Audio Extraction | `TranscriptAudioDelegate.kt` | Go | TODO |
| 9 | Long Video Chunking | `TranscriptChunkDelegate.kt` | Go | TODO |
| 10 | Transcript Cache | `TranscriptCacheDelegate.kt` | Go | TODO |
| 11 | JSON Parsing + Post-processing | `TranscriptJsonParser.kt` | Go | TODO |
| 12 | Subtitle Generation | `SubtitleGenerator.kt` | Go | TODO |

### AI Analysis

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 13 | Topic Classification | `TopicClassificationDelegate.kt` | Go | TODO |
| 14 | Explicit Transition Detection | `TopicExplicitTransitionDelegate.kt` | Go | TODO |
| 15 | Semantic Transition Detection | `TopicSemanticTransitionDelegate.kt` | Go | TODO |
| 16 | Music/Noise Filter | `TopicMusicFilterDelegate.kt` | Go | TODO |
| 17 | Speech Pattern Detection | `SpeechPatternDetector.kt` + `PatternDetectionDelegate.kt` | Go | TODO |
| 18 | Pattern Cache | `PatternCacheDelegate.kt` | Go | TODO |

### Highlight Detection

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 19 | Highlight Detector | `HighlightDetector.kt` | Go | TODO |
| 20 | Audio/Silence Analysis | `AudioAnalyzer.kt` | Go | TODO |
| 21 | Scene Analysis | `SceneAnalyzer.kt` | Go | TODO |
| 22 | Detection Strategy Selection | `DetectionStrategy.kt` | Go | TODO |

### Clip Generation

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 23 | Clip Cutting | `ClipProcessorService.kt` | Go | TODO |
| 24 | Clip Review | `ClipReviewService.kt` | Go | TODO |
| 25 | Text Overlay | `TextOverlayProcessor.kt` | Go | TODO |
| 26 | Video Overlay | `VideoOverlayProcessor.kt` | Go | TODO |
| 27 | Preview Generation | `PreviewProcessor.kt` | Go | TODO |
| 28 | Uniqueness Detection | `VideoUniquenessProcessor.kt` | Go | TODO |

### Shorts

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 29 | Shorts Generator | `ShortsGenerator.kt` | Go | TODO |
| 30 | Segment Detection | `ShortsSegmentDetector.kt` | Go | TODO |
| 31 | Vertical Video (9:16) | `VerticalVideoProcessor.kt` | Go | TODO |
| 32 | Shorts Cache | `ShortsCacheDelegate.kt` | Go | TODO |
| 33 | Parallel Generation (5 jobs) | `ShortsParallelGenerationDelegate.kt` | Go | TODO |
| 34 | Shorts Subtitles | `ShortsSubtitleDelegate.kt` | Go | TODO |
| 35 | Title Generation | `TitleGenerator.kt` | Go | TODO |

### Thumbnail

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 36 | Strategy — Centered | `CenteredThumbnailStrategy.kt` | Go | TODO |
| 37 | Strategy — Branded | `BrandedStrategy.kt` | Go | TODO |
| 38 | Strategy — BestFrame | `BestFrameStrategy.kt` | Go | TODO |
| 39 | Strategy — LongForm | `LongFormStrategy.kt` | Go | TODO |
| 40 | Strategy — Shorts | `ShortsThumbnailStrategy.kt` | Go | TODO |
| 41 | Strategy — Simple Overlay | `SimpleOverlayStrategy.kt` | Go | TODO |
| 42 | Strategy — Template | `TemplateStrategy.kt` | Go | TODO |
| 43 | Font Selection | `ThumbnailFontDelegate.kt` | Go | TODO |
| 44 | Color Customization | `ThumbnailColorsDelegate.kt` | Go | TODO |
| 45 | Presets (Shorts/LongForm/Live/React) | `ThumbnailPresets.kt` | Go | TODO |
| 46 | FFmpeg Text Renderer | `FFmpegTextRenderer.kt` + `FFmpegTextOverlayDelegate.kt` | Go | TODO |
| 47 | System Font Detection | `SystemFontDetector.kt` | Go | TODO |

### Upload & Auth

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 48 | YouTube Upload | `YouTubeUploader.kt` | Go | TODO |
| 49 | OAuth 2.0 | `OAuthAuthenticator.kt` + `AuthenticationDelegate.kt` | Go | TODO |
| 50 | Client Secret Management | `ClientSecretService.kt` | Go | TODO |
| 51 | Thumbnail Upload | `ThumbnailUploadDelegate.kt` | Go | TODO |
| 52 | Upload Progress | `UploadProgressDelegate.kt` | Go | TODO |
| 53 | Upload Validation | `UploadValidationDelegate.kt` | Go | TODO |
| 54 | Playlist Management | `PlaylistManagementDelegate.kt` | Go | TODO |
| 55 | Post-upload Metadata Update | `VideoManagementDelegate.kt` | Go | TODO |
| 56 | Comment Management | `CommentManagementDelegate.kt` | Go | TODO |
| 57 | Quota Tracking | `QuotaTrackingDelegate.kt` | Go | TODO |

### Scheduling & Queue

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 58 | Scheduling Service | `SchedulingService.kt` + `SchedulingDelegate.kt` | Go | TODO |
| 59 | Upload Queue | `UploadQueueService.kt` | Go | TODO |
| 60 | Privacy Status (public/private/unlisted) | `PrivacyStatus.kt` | Go | TODO |

### Channel Metadata

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 61 | Channel Management | `ChannelsViewModel.kt` + `ChannelImportService.kt` | Go | TODO |
| 62 | Channel Preferences | `ChannelPreferencesService.kt` | Go | TODO |
| 63 | Channel Logo | `ChannelLogoService.kt` | Go | TODO |
| 64 | Metadata Service | `ChannelMetadataService.kt` | Go | TODO |
| 65 | Browser Cookie Extraction | `BrowserCookieDelegate.kt` | Go | TODO |
| 66 | Metadata Cache | `MetadataCacheDelegate.kt` | Go | TODO |
| 67 | URL Normalization | `URLNormalizerDelegate.kt` | Go | TODO |
| 68 | Parallel Metadata Processing | `ParallelProcessorDelegate.kt` | Go | TODO |
| 69 | AI Prompt Generation | `PromptGeneratorDelegate.kt` | Go | TODO |
| 70 | Metadata History | `ChannelMetadataHistoryService.kt` | Go | TODO |

### Optimizer

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 71 | Audio Silence Detector | `AudioSilenceDetector.kt` | Go | TODO |
| 72 | Composite Silence Detector | `CompositeSilenceDetector.kt` | Go | TODO |
| 73 | Transcript Pause Detector | `TranscriptPauseDetector.kt` | Go | TODO |
| 74 | Speed Zones | `SpeedChainBuilder.kt` + `SpeedZoneDelegate.kt` | Go | TODO |
| 75 | Music Overlay | `MusicLibraryService.kt` | Go | TODO |
| 76 | Transition Filter | `TransitionFilterBuilder.kt` + `ComplexFilterBuilder.kt` | Go | TODO |
| 77 | Chunked Optimization | `ChunkedVideoOptimizer.kt` | Go | TODO |
| 78 | Optimization Cache | `OptimizationCacheDelegate.kt` | Go | TODO |

### Post-Optimization

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 79 | Video Splitting | `PostOptSplitDelegate.kt` | Go | TODO |
| 80 | Intro/Outro Insertion | `PostOptPatternDelegate.kt` | Go | TODO |
| 81 | Per-part Metadata | `PostOptMetadataDelegate.kt` | TS | TODO |
| 82 | Auto-thumbnail per part | `PostOptThumbnailDelegate.kt` | Go | TODO |
| 83 | Batch Upload Parts | `PostOptUploadDelegate.kt` | Go | TODO |

### Settings & Configuration

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 84 | Tool Path Configuration | `ToolsConfigDelegate.kt` | Go | TODO |
| 85 | Installation Service | `InstallationService.kt` | Go | TODO |
| 86 | Tool Path Auto-detection | `ToolPathDetectionDelegate.kt` | Go | TODO |
| 87 | Channel Styling | `ChannelStylingDelegate.kt` | Go+TS | TODO |
| 88 | Background Image Management | `ChannelBackgroundsDelegate.kt` | Go | TODO |
| 89 | Default Tags / Categories | `ChannelContentDelegate.kt` | Go | TODO |
| 90 | App Settings Persistence | `AppSettingsService.kt` | Go | TODO |

### Additional Features

| # | Feature | Kotlin Reference | Target | Status |
|---|---|---|---|---|
| 91 | Mass Update | `MassUpdateViewModel.kt` | TS | TODO |
| 92 | Update Checker | `UpdateCheckerService.kt` | Electron | TODO |
| 93 | Background Service / Queue Worker | `BackgroundService.kt` | Go | TODO |
| 94 | Queue View + Reorder + Cancel + Retry | `QueueViewModel.kt` | TS | TODO |
| 95 | Pipeline Workflow | All `*WorkflowDelegate.kt` steps | Go | TODO |

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

**Version**: 1.0.0 | **Ratified**: 2026-04-10 | **Last Amended**: 2026-04-10
