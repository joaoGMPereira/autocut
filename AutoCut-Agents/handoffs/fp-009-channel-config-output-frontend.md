---
feature: FP-009-ChannelConfig
stage: output
scope: frontend
status: approved
files_created:
  - AutoCut/apps/web/src/store/channelConfigStore.ts
  - AutoCut/apps/web/src/components/channels/ChannelConfigSheet.tsx
files_modified:
  - AutoCut/apps/web/src/app/channels/page.tsx
---

## Summary

FP-009 frontend implementation complete. Per-channel configuration drawer added to the Channels page.

### channelConfigStore.ts

Zustand store managing:
- `configs: Record<number, ChannelConfig>` — keyed by channel_id
- `blacklists: Record<number, MusicBlacklistItem[]>` — keyed by channel_id
- All six async actions (`fetchConfig`, `saveConfig`, `uploadLogo`, `fetchBlacklist`, `addBlacklistPattern`, `removeBlacklistPattern`) targeting `${goUrl}/api/channels/:id/...`
- `goUrl` is read from `useAppStore.getState()` at call time (not stored in the config store)

### ChannelConfigSheet.tsx

480px right-side Sheet with four tabs:

| Tab | Fields |
|-----|--------|
| Branding | gradient start/end (color picker + hex input), font family, text color picker, logo upload/remove |
| Defaults | default_category_id (number), default_tags (textarea, comma-separated), max_highlights (number, nullable) |
| Processing | anti_duplicate_enabled (Checkbox), branding_logo_enabled (Checkbox) |
| Music | music blacklist — list with per-item delete, add-pattern input |

Notes:
- No `Switch` component exists in this project; used `Checkbox` from `@/components/ui/checkbox` for boolean toggles
- No `Textarea` shadcn component exists; used a plain `<textarea>` styled to match Input
- Music tab fetches blacklist on mount via `useEffect`
- Save button at sheet footer calls `saveConfig` with accumulated local state

### page.tsx changes

- Added `configChannel: Channel | null` state
- Added `Settings2` (lucide-react) gear + "Configure" outline button per channel row
- Mounted `<ChannelConfigSheet>` at the bottom of the return tree, controlled by `configChannel`

### TypeScript

`pnpm tsc --noEmit` exits 0, no errors.
