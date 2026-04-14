# Quickstart: Download Pipeline — Validation & Hardening

How to run and manually verify this feature locally.

---

## Prerequisites

```bash
which yt-dlp          # must be on PATH
which ffmpeg          # required by yt-dlp for mp4 merge
go version            # 1.23+
node --version        # 20+
pnpm --version        # 9+
```

---

## Start the stack

**Terminal A — Go server**:
```bash
cd AutoCut/server
CGO_ENABLED=0 go run ./cmd/server
# Expected: listening on http://127.0.0.1:4070
```

**Terminal B — Next.js frontend**:
```bash
cd AutoCut/apps/web
pnpm dev
# Opens http://localhost:3000
```

**Health check**:
```bash
curl http://127.0.0.1:4070/api/health
# Expected: {"status":"ok"}
```

---

## Happy path: download with live progress

1. Open `http://localhost:3000` → click **New Pipeline Run**
2. Paste a YouTube URL (short video recommended, e.g. < 2 min)
3. Watch `DownloadInfoCard`:
   - Progress bar must advance (not frozen at 0%)
   - Speed (KB/s) and ETA must appear on most updates
4. After completion: UI transitions to the Mode selection step

**Verify via DB**:
```bash
sqlite3 ~/.autocut/autocut.db \
  "SELECT id, state, video_path, video_title, duration_sec FROM pipeline_runs ORDER BY id DESC LIMIT 1;"
# Expected: state=WAITING_MODE, video_path non-empty, video_title non-empty, duration_sec > 0
```

---

## Cancel mid-download

1. Start a download of a long video (> 5 min)
2. Within 5 seconds, click **Cancel**
3. Verify:
   - UI transitions to cancelled state
   - Background yt-dlp process terminates (check `ps aux | grep yt-dlp` — should be gone within 3 s)
   - DB: `state = 'CANCELLED'`

---

## Invalid URL rejection

```bash
curl -s -X POST http://127.0.0.1:4070/api/pipeline/runs \
  -H 'Content-Type: application/json' | jq '.run_id' | xargs -I{} \
  curl -s -X POST http://127.0.0.1:4070/api/pipeline/runs/{}/advance \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://vimeo.com/12345"}'
# Expected: {"error":"unsupported platform: ..."}  HTTP 400
```

---

## Run validation scripts

```bash
# Backend
cd AutoCut
bash scripts/validate/validate-02.sh

# Frontend
bash scripts/validate/validate-02-fe.sh
```

Both must exit 0.

---

## Build check

```bash
cd AutoCut/server
CGO_ENABLED=0 go build ./...

cd AutoCut/apps/web
pnpm tsc --noEmit
```

Both must succeed with no errors.
