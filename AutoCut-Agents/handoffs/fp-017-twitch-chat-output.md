---
feature: FP-017-TwitchChatAnalysis
stage: output
scope: full
status: approved
files_modified:
  - AutoCut/server/internal/downloader/twitch.go
  - AutoCut/server/internal/api/handlers/download_handler.go
  - AutoCut/server/internal/ai/chat_activity_detector.go
  - AutoCut/server/internal/api/handlers/ai_handler.go
  - AutoCut/apps/web/src/store/downloadStore.ts
  - AutoCut/apps/web/src/store/highlightStore.ts
  - AutoCut/apps/web/src/components/pipeline/steps/DownloadStep.tsx
  - AutoCut/apps/web/src/components/highlights/HighlightList.tsx
---

# FP-017 — Twitch Chat Analysis Highlights

## Summary

Full implementation of Twitch chat download + analysis integration across backend and frontend.

## Backend Changes

### `server/internal/downloader/twitch.go`
- Replaced existing `DownloadChat(vodID, outDir string)` (no callers) with new signature:
  `DownloadChat(ctx context.Context, videoURL, chatOutputPath string) error`
- New behaviour:
  - Extracts video ID from URL via existing `extractVodID`
  - Creates parent directory of `chatOutputPath` with `os.MkdirAll`
  - If `TwitchDownloaderCLI` is not available on PATH: logs `slog.Warn`, writes `[]` to `chatOutputPath`, returns nil (graceful)
  - If available: runs `TwitchDownloaderCLI chatdownload --id {id} --output {path} --format JSON`
  - Respects `ctx.Done()` before executing
- Added `"context"` import

### `server/internal/api/handlers/download_handler.go`
- Added `DownloadChat bool \`json:"download_chat"\`` field to `downloadRequest`
- Added `"context"` import; threaded `context.Background()` into `runDownload` goroutine
- After a successful Twitch VOD download, if `DownloadChat == true`:
  1. Creates `{output_dir}/chat/` directory
  2. Calls `h.twDl.DownloadChat(ctx, req.URL, {output_dir}/chat/{videoID}.json)`
  3. On success: emits SSE event `{"type":"chat_downloaded","data":{"chat_path":"<path>"}}`
  4. On failure: logs `slog.Warn` (non-fatal, does not abort the download job)

### `server/internal/ai/chat_activity_detector.go`
- No changes required — already handles empty arrays gracefully (lines 75-78: `return nil, nil` when `len(messages) == 0`)

### `server/internal/api/handlers/ai_handler.go`
- No changes required — already fully implements the `chat` strategy:
  - Reads `ChatJSONPath` from request
  - Calls `ai.NewChatActivityDetector().DetectHighlights(ctx, req.ChatJSONPath)` in a goroutine
  - Skips with `slog.Warn` when `chat_json_path` is empty

## Frontend Changes

### `apps/web/src/store/downloadStore.ts`
- Added `download_chat?: boolean` to `DownloadPayload` interface
- Extended `SseEvent` union type to include `{ type: 'chat_downloaded'; data: { chat_path: string } }`
- Added SSE handler for `chat_downloaded`: stores `chat_path` in `job.result` so consumers can read it
- Updated `done` handler to merge with existing `job.result` (preserves `chat_path` set before `done` fires)

### `apps/web/src/store/highlightStore.ts`
- No changes required — already has `chatJsonPath: string`, `setChatJsonPath`, and passes `chat_json_path` in the analyze request body when the `chat` strategy is selected

### `apps/web/src/components/pipeline/steps/DownloadStep.tsx`
- Added local state `downloadChat: boolean` (default `false`)
- Added "Download Chat" checkbox to the options row, **only visible when `platform === 'twitch'`**
- Passes `download_chat: platform === 'twitch' ? downloadChat : undefined` to `startDownload`

### `apps/web/src/components/highlights/HighlightList.tsx`
- Imported `MessageCircle` from `lucide-react`
- In `HighlightRow` strategy badges: when `s === 'chat'`, renders `<MessageCircle className="h-2.5 w-2.5" />` inline before the label text

## Verification

```
# Go
cd AutoCut/server && CGO_ENABLED=0 go build ./...   # exits 0

# TypeScript
cd AutoCut/apps/web && pnpm tsc --noEmit             # exits 0
```

Both checks pass with zero errors.

## Integration Flow

1. User selects Twitch platform in DownloadStep, checks "Download Chat"
2. `POST /api/download` with `download_chat: true`
3. Backend downloads VOD, then downloads chat to `{output_dir}/chat/{videoID}.json`
4. SSE emits `chat_downloaded` → frontend stores `chat_path` in job result
5. User navigates to Highlights, selects "chat" strategy
6. `HighlightStrategySelector` shows the Chat JSON Path input (pre-fillable from `chat_path`)
7. `POST /api/analyze` with `strategies: ["chat"], chat_json_path: "<path>"`
8. Backend runs `ChatActivityDetector.DetectHighlights`, returns highlights
9. Highlights with `strategies: ["chat"]` display with a `MessageCircle` icon badge
