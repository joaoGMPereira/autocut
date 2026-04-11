---
feature: FP-010-OAuthMultiProfile
stage: output
scope: full
status: approved
files_created:
  - AutoCut/server/internal/api/handlers/oauth_handler.go
  - AutoCut/apps/web/src/store/oauthStore.ts
  - AutoCut/apps/web/src/components/settings/OAuthProfilesSection.tsx
  - AutoCut/apps/web/src/components/settings/ChannelAuthSection.tsx
files_modified:
  - AutoCut/server/internal/api/router.go
  - AutoCut/server/cmd/server/main.go
  - AutoCut/server/cmd/autocut/main.go
  - AutoCut/apps/web/src/app/settings/page.tsx
---

## Summary

FP-010 implements full OAuth multi-profile management: backend CRUD for `oauth_client_secrets`
and the OAuth2 authorization flow, plus a frontend UI to manage profiles and authorize channels.

## Backend

### `oauth_handler.go`

New `OAuthHandler` struct wired with `OAuthClientSecretRepo` and `ChannelRepo`.

| Method | Route | Notes |
|---|---|---|
| `ListProfiles` | `GET /api/oauth/profiles` | Returns id/name/project_id/is_default — never client_id/client_secret |
| `UploadProfile` | `POST /api/oauth/profiles` | Accepts multipart with `name` + `file` (client_secret.json); parses `installed` or `web` key |
| `DeleteProfile` | `DELETE /api/oauth/profiles/{id}` | |
| `SetDefaultProfile` | `POST /api/oauth/profiles/{id}/set-default` | Uses `OAuthClientSecretRepo.SetDefault` (transactional) |
| `InitOAuthFlow` | `POST /api/channels/{id}/auth` | Builds Google OAuth2 URL; returns 400 `no_oauth_profile` if channel has no profile |
| `HandleOAuthCallback` | `POST /api/channels/{id}/auth/callback` | Exchanges code via `oauth2.googleapis.com/token`; saves tokens with `UpdateTokens` |

Token exchange uses `net/http` only (no external OAuth lib). `expires_in` (seconds) is converted
to unix milliseconds for consistency with the rest of the codebase.

### Router + main changes

- `router.go`: added `oauthH *handlers.OAuthHandler` parameter and 6 new route registrations.
- `cmd/server/main.go`: `oauthH := handlers.NewOAuthHandler(db)` added before `NewRouter` call.
- `cmd/autocut/main.go`: same addition (second entrypoint in the repo).

Build verified: `CGO_ENABLED=0 go build ./...` exits 0.

## Frontend

### `oauthStore.ts`

Zustand store with `goUrl` passed at call-site (matches channelStore pattern):
`fetchProfiles`, `uploadProfile`, `deleteProfile`, `setDefault`, `initOAuth`, `submitCode`.

### `OAuthProfilesSection.tsx`

Settings section listing OAuth profiles with:
- Name, project_id, Default badge
- "Set Default" button (hidden when already default)
- "Delete" button with per-row loading state
- Upload form: name input + `.json` file input + Upload button with error display

### `ChannelAuthSection.tsx`

Settings section listing all channels with their auth status:
- `AuthBadge`: green Authorized / yellow Token Expired / grey Not Authorized
- "Authorize" button calls `initOAuth`; on success opens a Dialog
- Dialog shows the auth URL in a read-only textarea + "Open" button (`window.open`)
- Below URL: code input + "Confirm" button calls `submitCode`; errors shown inline
- On success: dialog closes, channels refetched

### `settings/page.tsx`

`OAuthProfilesSection` and `ChannelAuthSection` inserted between existing `ChannelsSection`
and `ToolsSection`. Both are `'use client'` components; the page remains a Server Component.

TS check: `pnpm tsc --noEmit` exits 0, no errors.
