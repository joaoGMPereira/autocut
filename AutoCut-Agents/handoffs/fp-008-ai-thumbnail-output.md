---
feature: FP-008-AIThumbnail
stage: output
scope: full
status: approved
files_modified:
  - AutoCut/server/internal/thumbnail/strategies.go
  - AutoCut/server/internal/api/handlers/thumbnail_handler.go
  - AutoCut/apps/web/src/app/thumbnail/page.tsx
---

# FP-008 — AI-Generated Thumbnail

## Summary

Added an `ai` strategy to the thumbnail pipeline (backend + frontend). The strategy extracts a video
frame at 30 % of duration, sends it to Ollama's vision model (llava) with a CTR-optimised prompt,
uses the AI-generated text as the overlay title, then renders a branded thumbnail. Falls back to
`"Epic Moment"` if Ollama is unreachable or returns an empty response.

## Backend changes

### `server/internal/thumbnail/strategies.go`

- Added `AIThumbnailConfig` struct with fields: `VideoPath`, `CustomPrompt`, `OllamaURL`,
  `FontPath`, `FontColor`, `FontSize`, `Width`, `Height`. `OllamaURL` defaults to the
  `OLLAMA_URL` env var, then `"http://localhost:11434"`.
- Added `ollamaVisionRequest` / `ollamaVisionResponse` types for the non-streaming Ollama payload.
- Added `(g *ThumbnailGenerator) GenerateAIThumbnail(cfg AIThumbnailConfig, output string) error`
  — extracts frame, calls Ollama, renders branded thumbnail.
- Added `(g *ThumbnailGenerator) callOllamaForTitle(ollamaURL, prompt, framePath string) string`
  — self-contained HTTP helper; uses llava with base64 image when frame available, falls back to
  llama3 text-only; returns safe fallback on any error; logs via `slog`.

### `server/internal/api/handlers/thumbnail_handler.go`

- Added `CustomPrompt string \`json:"custom_prompt"\`` field to `thumbnailRequest`.
- Added `"ai"` case in `PostThumbnail`'s strategy switch: constructs `AIThumbnailConfig` and
  calls `h.gen.GenerateAIThumbnail(cfg, req.Output)`.

### Build

`CGO_ENABLED=0 go build ./...` — clean, no errors.

## Frontend changes

### `apps/web/src/app/thumbnail/page.tsx`

- Extended `Strategy` type to `'branded' | 'centered' | 'template' | 'ai'`.
- Added `customPrompt` local state (`useState<string>('')`).
- Added `'ai'` to the RadioGroup options array; label rendered as `"AI-Generated"`.
- Added AI-specific panel shown when `strategy === 'ai'`:
  - `<textarea>` for custom Ollama prompt with placeholder text.
  - Info note about llava/llama3 fallback behaviour.
- Added `custom_prompt` to fetch body when `strategy === 'ai'` and prompt is non-empty.

### TypeScript

`pnpm tsc --noEmit` — clean, no errors.
