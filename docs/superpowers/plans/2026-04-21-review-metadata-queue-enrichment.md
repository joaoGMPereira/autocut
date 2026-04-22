# Review Metadata + Queue Enrichment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add thumbnail-type badge to StepReviewMetadata, enrich the Queue page with thumbnails/channel filter, and fix the upload-queue backend (thumbnail correlation, metadata.json, MadeForKids=false).

**Architecture:** Three independent layers: (1) pure frontend badge, (2) backend queue CRUD handler + frontend rich cards, (3) backend queue-on-upload-confirm replacing the stub. Backend tasks must land before frontend queue tasks because the frontend queue page depends on real API responses.

**Tech Stack:** Go 1.23, `net/http` stdlib, `database/sql` + `modernc.org/sqlite`, TypeScript 5.8 / React 19, Next.js 16, Zustand, shadcn/ui, Tailwind CSS.

---

## Critical context before starting

- `server/internal/uploader/types.go`, `metadata.go`, `auth.go`, `quota.go`, `strategy.go`, `youtube.go` are **all stub files** (4-line TODO bodies). Only touch `types.go`, `metadata.go`, and create `storage.go`.
- `server/internal/api/handlers/queue_handler.go` is a **stub** — 4-line TODO body.
- No queue routes are registered in `router.go`.
- `apps/web/src/app/queue/page.tsx` renders `<NotImplemented>` — must be replaced.
- `StateWaitingUploadConfirm` in `service.go` calls `runUploadStub()` which just sleeps 100ms and advances to DONE — this is the placeholder we replace.
- `UploadRepo.CreateQueued` current signature: `(ctx, channelID int64, videoPath, metadataJSON, uploadConfigJSON string, queueOrder int)` — missing `clipID` and `thumbnailPath`.
- `QueueRow` struct has no `Title` or `ThumbnailPath` fields — we add a new `QueueItemWithTitle` struct and `ListQueueWithTitle` method.

---

## Task 1: Define uploader types

**Files:**
- Modify: `server/internal/uploader/types.go`

- [ ] **Step 1: Write the test**

Create `server/internal/uploader/types_test.go`:

```go
package uploader_test

import (
	"testing"
)

func TestVideoMetadataDefaults(t *testing.T) {
	meta := VideoMetadata{
		Title:       "Test",
		Description: "Desc",
		Tags:        []string{"a", "b"},
		CategoryID:  "22",
		Privacy:     "private",
		MadeForKids: false,
		Language:    "pt-BR",
	}
	if meta.MadeForKids {
		t.Fatal("MadeForKids must default to false")
	}
}
```

Run: `cd server && go test ./internal/uploader/... -run TestVideoMetadataDefaults`
Expected: compile error — `VideoMetadata` undefined.

- [ ] **Step 2: Replace types.go stub**

```go
package uploader

// VideoMetadata holds all YouTube upload metadata fields.
// MadeForKids is always false — never read from channel config.
type VideoMetadata struct {
	ChannelID         int64    `json:"channel_id"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Tags              []string `json:"tags"`
	CategoryID        string   `json:"category_id"`
	Privacy           string   `json:"privacy"`
	MadeForKids       bool     `json:"made_for_kids"`
	Language          string   `json:"language"`
}
```

- [ ] **Step 3: Run test**

Run: `cd server && go test ./internal/uploader/... -run TestVideoMetadataDefaults`
Expected: PASS

- [ ] **Step 4: Build check**

Run: `cd server && go build ./...`
Expected: success (other stub files still compile as empty packages)

- [ ] **Step 5: Commit**

```bash
git add server/internal/uploader/types.go server/internal/uploader/types_test.go
git commit -m "feat(uploader): define VideoMetadata struct with MadeForKids=false"
```

---

## Task 2: BuildMetadata function

**Files:**
- Modify: `server/internal/uploader/metadata.go`

- [ ] **Step 1: Write the test**

Create `server/internal/uploader/metadata_test.go`:

```go
package uploader_test

import (
	"strings"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

func TestBuildMetadata_MadeForKidsAlwaysFalse(t *testing.T) {
	clip := database.PipelineClip{
		Title:       "Clip title",
		Description: "desc",
		Tags:        "tag1,tag2",
	}
	cfg := database.ChannelConfig{
		ChannelID:        1,
		MadeForKids:      true, // even when channel config says true
		DefaultCategoryID: 22,
		DefaultTags:      "extra",
	}
	meta := BuildMetadata(clip, cfg)
	if meta.MadeForKids {
		t.Fatal("BuildMetadata must always return MadeForKids=false")
	}
	if meta.Title != "Clip title" {
		t.Fatalf("expected title %q, got %q", "Clip title", meta.Title)
	}
	if !containsTag(meta.Tags, "tag1") {
		t.Fatal("clip tags not included")
	}
	if !containsTag(meta.Tags, "extra") {
		t.Fatal("channel default tags not included")
	}
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.TrimSpace(t) == tag {
			return true
		}
	}
	return false
}
```

Run: `cd server && go test ./internal/uploader/... -run TestBuildMetadata`
Expected: FAIL — `BuildMetadata` undefined.

- [ ] **Step 2: Replace metadata.go stub**

```go
package uploader

import (
	"fmt"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// BuildMetadata merges clip + channel config into a VideoMetadata value.
// MadeForKids is always false regardless of channel config.
func BuildMetadata(clip database.PipelineClip, cfg database.ChannelConfig) VideoMetadata {
	tags := splitTags(clip.Tags)
	if cfg.DefaultTags != "" {
		tags = append(tags, splitTags(cfg.DefaultTags)...)
	}
	return VideoMetadata{
		ChannelID:   cfg.ChannelID,
		Title:       clip.Title,
		Description: clip.Description,
		Tags:        tags,
		CategoryID:  fmt.Sprintf("%d", cfg.DefaultCategoryID),
		Privacy:     "private",
		MadeForKids: false, // hardcoded — never from channel config
		Language:    "pt-BR",
	}
}

// splitTags splits a comma-separated tag string, trimming whitespace.
func splitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
```

- [ ] **Step 3: Run test**

Run: `cd server && go test ./internal/uploader/... -run TestBuildMetadata`
Expected: PASS

- [ ] **Step 4: Build check**

Run: `cd server && go build ./...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add server/internal/uploader/metadata.go server/internal/uploader/metadata_test.go
git commit -m "feat(uploader): implement BuildMetadata with hardcoded MadeForKids=false"
```

---

## Task 3: QueueStorage — SaveToQueue

**Files:**
- Modify: `server/internal/uploader/storage.go` (currently a stub)

- [ ] **Step 1: Write the test**

Create `server/internal/uploader/storage_test.go`:

```go
package uploader_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveToQueue_CreatesFilesAndMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	qs := NewQueueStorage(tmpDir)

	// Create dummy source files
	srcVideo := filepath.Join(tmpDir, "clip.mp4")
	srcThumb := filepath.Join(tmpDir, "thumb.jpg")
	if err := os.WriteFile(srcVideo, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcThumb, []byte("thumb"), 0o644); err != nil {
		t.Fatal(err)
	}

	meta := VideoMetadata{
		ChannelID:   3,
		Title:       "Test clip",
		Description: "desc",
		Tags:        []string{"tag1"},
		CategoryID:  "22",
		Privacy:     "private",
		MadeForKids: false,
		Language:    "pt-BR",
	}

	qVideo, qThumb, err := qs.SaveToQueue(srcVideo, srcThumb, meta)
	if err != nil {
		t.Fatalf("SaveToQueue failed: %v", err)
	}

	// video + thumbnail must exist at returned paths
	if _, err := os.Stat(qVideo); err != nil {
		t.Fatalf("queued video not found: %v", err)
	}
	if _, err := os.Stat(qThumb); err != nil {
		t.Fatalf("queued thumbnail not found: %v", err)
	}

	// metadata.json must exist in same dir
	metaPath := filepath.Join(filepath.Dir(qVideo), "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("metadata.json not found: %v", err)
	}
	var got VideoMetadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse metadata.json: %v", err)
	}
	if got.Title != "Test clip" {
		t.Fatalf("metadata title mismatch: %q", got.Title)
	}
	if got.MadeForKids {
		t.Fatal("metadata.json must have made_for_kids=false")
	}
}

func TestSaveToQueue_NoThumbnail(t *testing.T) {
	tmpDir := t.TempDir()
	qs := NewQueueStorage(tmpDir)

	srcVideo := filepath.Join(tmpDir, "clip.mp4")
	if err := os.WriteFile(srcVideo, []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}

	qVideo, qThumb, err := qs.SaveToQueue(srcVideo, "", VideoMetadata{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(qVideo); err != nil {
		t.Fatal("video must be saved even without thumbnail")
	}
	if qThumb != "" {
		t.Fatalf("qThumb should be empty when no thumbnail provided, got %q", qThumb)
	}
}
```

Run: `cd server && go test ./internal/uploader/... -run TestSaveToQueue`
Expected: FAIL — `NewQueueStorage` undefined.

- [ ] **Step 2: Replace storage.go stub**

```go
package uploader

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// QueueStorage manages the upload_queue directory on disk.
type QueueStorage struct {
	dataDir string
}

// NewQueueStorage creates a QueueStorage rooted at dataDir.
func NewQueueStorage(dataDir string) *QueueStorage {
	return &QueueStorage{dataDir: dataDir}
}

// SaveToQueue copies video + optional thumbnail to a new upload_queue subdirectory,
// writes metadata.json, and returns the queued video path and thumbnail path.
// If thumbnailPath is empty, the thumbnail is skipped and queuedThumb is "".
func (q *QueueStorage) SaveToQueue(videoPath, thumbnailPath string, metadata VideoMetadata) (queuedVideo, queuedThumb string, err error) {
	ts := time.Now().UnixMilli()
	rnd := rand.Int63n(9000) + 1000 //nolint:gosec
	dirName := fmt.Sprintf("%d_%d", ts, rnd)
	queueDir := filepath.Join(q.dataDir, "upload_queue", dirName)
	if err := os.MkdirAll(queueDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create queue dir: %w", err)
	}

	// Copy video
	dstVideo := filepath.Join(queueDir, "video.mp4")
	if err := copyFile(videoPath, dstVideo); err != nil {
		return "", "", fmt.Errorf("copy video: %w", err)
	}

	// Copy thumbnail (optional — non-fatal if source missing)
	dstThumb := ""
	if thumbnailPath != "" {
		candidate := filepath.Join(queueDir, "thumbnail.jpg")
		if cpErr := copyFile(thumbnailPath, candidate); cpErr == nil {
			dstThumb = candidate
		}
	}

	// Write metadata.json
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(queueDir, "metadata.json"), metaBytes, 0o644); err != nil {
		return "", "", fmt.Errorf("write metadata.json: %w", err)
	}

	return dstVideo, dstThumb, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return out.Close()
}
```

- [ ] **Step 3: Run tests**

Run: `cd server && go test ./internal/uploader/... -run TestSaveToQueue`
Expected: PASS (both sub-tests)

- [ ] **Step 4: Build check**

Run: `cd server && go build ./...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add server/internal/uploader/storage.go server/internal/uploader/storage_test.go
git commit -m "feat(uploader): implement QueueStorage.SaveToQueue — creates upload_queue dir, writes metadata.json"
```

---

## Task 4: Fix CreateQueued + add ListQueueWithTitle

**Files:**
- Modify: `server/internal/database/upload_repo.go`

- [ ] **Step 1: Write failing tests**

Create `server/internal/database/upload_repo_queue_test.go`:

```go
package database_test

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Minimal schema for uploads + pipeline_clips
	_, err = db.Exec(`
		CREATE TABLE uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			clip_id INTEGER NOT NULL DEFAULT 0,
			channel_id INTEGER NOT NULL DEFAULT 0,
			youtube_id TEXT NOT NULL DEFAULT '',
			youtube_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'queued',
			scheduled_at INTEGER,
			uploaded_at INTEGER,
			error TEXT NOT NULL DEFAULT '',
			video_type TEXT NOT NULL DEFAULT 'long_form',
			local_video_path TEXT,
			local_thumbnail_path TEXT,
			metadata_json TEXT NOT NULL DEFAULT '',
			upload_config_json TEXT NOT NULL DEFAULT '',
			original_video_name TEXT NOT NULL DEFAULT '',
			source_video_url TEXT NOT NULL DEFAULT '',
			source_clip_url TEXT NOT NULL DEFAULT '',
			shorts_generated INTEGER NOT NULL DEFAULT 0,
			shorts_generated_at INTEGER,
			queue_order INTEGER NOT NULL DEFAULT 0,
			publish_at TEXT,
			created_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE pipeline_clips (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL DEFAULT '',
			thumbnail_path TEXT NOT NULL DEFAULT '',
			duration_sec REAL NOT NULL DEFAULT 0
		);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestCreateQueued_SavesThumbnailPath(t *testing.T) {
	db := openTestDB(t)
	repo := NewUploadRepo(db, slog.Default())
	ctx := context.Background()

	id, err := repo.CreateQueued(ctx, 1, 42, "/queue/video.mp4", "/queue/thumb.jpg", `{"title":"t"}`, "{}", 1)
	if err != nil {
		t.Fatalf("CreateQueued: %v", err)
	}
	u, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !u.LocalThumbnailPath.Valid || u.LocalThumbnailPath.String != "/queue/thumb.jpg" {
		t.Fatalf("thumbnail path not saved: %v", u.LocalThumbnailPath)
	}
	if u.ClipID != 42 {
		t.Fatalf("clip_id not saved: %d", u.ClipID)
	}
}

func TestListQueueWithTitle_JoinsThumbnail(t *testing.T) {
	db := openTestDB(t)
	repo := NewUploadRepo(db, slog.Default())
	ctx := context.Background()

	// Insert clip
	res, _ := db.Exec(`INSERT INTO pipeline_clips (run_id, title, thumbnail_path, duration_sec) VALUES (1, 'My Clip', '/clips/thumb.jpg', 55)`)
	clipID, _ := res.LastInsertId()

	_, err := repo.CreateQueued(ctx, 1, clipID, "/q/video.mp4", "/q/thumb.jpg", `{"title":"My Clip"}`, "{}", 1)
	if err != nil {
		t.Fatalf("CreateQueued: %v", err)
	}

	items, err := repo.ListQueueWithTitle(ctx)
	if err != nil {
		t.Fatalf("ListQueueWithTitle: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Title != "My Clip" {
		t.Fatalf("title not joined: %q", items[0].Title)
	}
	if items[0].ThumbnailPath != "/clips/thumb.jpg" {
		t.Fatalf("thumbnail path not joined: %q", items[0].ThumbnailPath)
	}
	if items[0].VideoType != "short" {
		t.Fatalf("expected video_type=short for 55s clip, got %q", items[0].VideoType)
	}
}
```

Run: `cd server && go test ./internal/database/... -run TestCreateQueued_SavesThumbnailPath`
Expected: FAIL — wrong number of arguments to CreateQueued.

- [ ] **Step 2: Update CreateQueued signature**

In `upload_repo.go`, find the `CreateQueued` function (around line 319) and replace it:

```go
// CreateQueued inserts a new upload with status='queued' for a specific clip.
// clipID links the upload to its pipeline_clips row (0 if no clip association).
// thumbnailPath is saved to local_thumbnail_path; empty string is allowed.
func (r *UploadRepo) CreateQueued(ctx context.Context, channelID, clipID int64, videoPath, thumbnailPath, metadataJSON, uploadConfigJSON string, queueOrder int) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO uploads (
			clip_id, channel_id, youtube_id, youtube_url, status,
			error, video_type, local_video_path, local_thumbnail_path,
			metadata_json, upload_config_json,
			original_video_name, source_video_url, source_clip_url,
			shorts_generated, queue_order, created_at
		) VALUES (?, ?, '', '', 'queued', '', 'long_form', ?, ?, ?, ?, '', '', '', 0, ?, ?)
	`, clipID, channelID, videoPath, thumbnailPath, metadataJSON, uploadConfigJSON, queueOrder, now)
	if err != nil {
		return 0, fmt.Errorf("create queued upload: %w", err)
	}
	return res.LastInsertId()
}
```

- [ ] **Step 3: Add QueueItemWithTitle struct + ListQueueWithTitle method**

Add after the `QueueRow` struct (around line 288):

```go
// QueueItemWithTitle is the rich projection used by the queue list API.
// It joins pipeline_clips to include title, thumbnail_path, and video_type.
type QueueItemWithTitle struct {
	ID            int64
	ClipID        int64
	ChannelID     int64
	Status        string
	VideoPath     string
	QueueOrder    int
	PublishAt     sql.NullString
	YoutubeID     string
	YoutubeURL    string
	Error         string
	CreatedAt     int64
	Title         string
	ThumbnailPath string
	VideoType     string
}

// ListQueueWithTitle returns uploads with status in
// (queued, running, failed, error, uploaded) joined with pipeline_clips for
// title, thumbnail_path, and video_type.
func (r *UploadRepo) ListQueueWithTitle(ctx context.Context) ([]QueueItemWithTitle, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.clip_id, u.channel_id, u.status,
		       COALESCE(u.local_video_path, '') as video_path,
		       u.queue_order, u.publish_at,
		       u.youtube_id, u.youtube_url, u.error, u.created_at,
		       COALESCE(c.title, '') as title,
		       COALESCE(c.thumbnail_path, '') as thumbnail_path,
		       CASE WHEN COALESCE(c.duration_sec, 999) <= 60 THEN 'short' ELSE 'long_form' END as video_type
		FROM uploads u
		LEFT JOIN pipeline_clips c ON c.id = u.clip_id
		WHERE u.status IN ('queued', 'running', 'failed', 'error', 'uploaded')
		ORDER BY u.queue_order ASC, u.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list queue with title: %w", err)
	}
	defer rows.Close()

	var result []QueueItemWithTitle
	for rows.Next() {
		var q QueueItemWithTitle
		if err := rows.Scan(
			&q.ID, &q.ClipID, &q.ChannelID, &q.Status, &q.VideoPath,
			&q.QueueOrder, &q.PublishAt,
			&q.YoutubeID, &q.YoutubeURL, &q.Error, &q.CreatedAt,
			&q.Title, &q.ThumbnailPath, &q.VideoType,
		); err != nil {
			return nil, fmt.Errorf("scan queue item with title: %w", err)
		}
		result = append(result, q)
	}
	return result, rows.Err()
}
```

- [ ] **Step 4: Run tests**

Run: `cd server && go test ./internal/database/... -run "TestCreateQueued|TestListQueueWithTitle"`
Expected: PASS (both tests)

- [ ] **Step 5: Build check** (may fail — callers of old CreateQueued signature need updating)

Run: `cd server && go build ./...`
Expected: compile error in `pipeline/service.go` or wherever CreateQueued is called (caller used old 6-arg signature).

Note: The only existing caller of `CreateQueued` is the TODO placeholder in `StateWaitingUploadConfirm` — we fix that in Task 5.

- [ ] **Step 6: Commit**

```bash
git add server/internal/database/upload_repo.go server/internal/database/upload_repo_queue_test.go
git commit -m "feat(upload_repo): fix CreateQueued (clipID+thumbnailPath), add ListQueueWithTitle with thumbnail join"
```

---

## Task 5: Pipeline service — real upload-confirm

**Files:**
- Modify: `server/internal/pipeline/service.go`

- [ ] **Step 1: Add uploadRepo + queueStorage fields to Service struct**

In `service.go`, find the `Service` struct (around line 34). Add two fields:

```go
type Service struct {
	repo           *database.PipelineRunRepo
	historyRepo    *database.URLHistoryRepo
	highlightRepo  *database.PipelineHighlightRepo
	clipRepo       *database.PipelineClipRepo
	channelCfgRepo *database.ChannelConfigRepo
	settingRepo    *database.AppSettingRepo
	uploadRepo     *database.UploadRepo      // NEW
	queueStorage   *uploader.QueueStorage    // NEW
	hub            *hub.SSEHub
	ytDl           *downloader.YouTubeDownloader
	twDl           *downloader.TwitchDownloader
	dataDir        string
	log            *slog.Logger
	cancelMap      sync.Map
	metaGenerator  metaGeneratorIface
}
```

- [ ] **Step 2: Update NewService to accept the new deps**

Add the two new parameters to `NewService` (around line 97):

```go
func NewService(
	repo *database.PipelineRunRepo,
	historyRepo *database.URLHistoryRepo,
	highlightRepo *database.PipelineHighlightRepo,
	clipRepo *database.PipelineClipRepo,
	channelCfgRepo *database.ChannelConfigRepo,
	h *hub.SSEHub,
	ytDl *downloader.YouTubeDownloader,
	twDl *downloader.TwitchDownloader,
	dataDir string,
	settingRepo *database.AppSettingRepo,
	uploadRepo *database.UploadRepo,
	queueStorage *uploader.QueueStorage,
) *Service {
	return &Service{
		repo:           repo,
		historyRepo:    historyRepo,
		highlightRepo:  highlightRepo,
		clipRepo:       clipRepo,
		channelCfgRepo: channelCfgRepo,
		settingRepo:    settingRepo,
		uploadRepo:     uploadRepo,
		queueStorage:   queueStorage,
		hub:            h,
		ytDl:           ytDl,
		twDl:           twDl,
		dataDir:        dataDir,
		log:            slog.With("component", "pipeline.service"),
	}
}
```

Also add the `uploader` import at the top of the file:
```go
"github.com/joaoGMPereira/autocut/server/internal/uploader"
```

- [ ] **Step 3: Replace the StateWaitingUploadConfirm case**

Find the `StateWaitingUploadConfirm` case in the `Advance` method (around line 404). Replace:

```go
case StateWaitingUploadConfirm:
    if err := s.repo.AdvanceState(ctx, id, StateWaitingUploadConfirm, StateUploading); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
                return AdvanceResult{State: curr.State}, nil
            }
        }
        return AdvanceResult{}, fmt.Errorf("advance run %d: %w", id, err)
    }
    s.hub.Publish(jobKey, hub.SSEEvent{
        Type: "state_changed",
        Data: stateChangedPayload{RunID: id, State: StateUploading},
    })
    go s.runUploadStub(id)
    return AdvanceResult{State: StateUploading}, nil
```

With:

```go
case StateWaitingUploadConfirm:
    return s.confirmUpload(ctx, id, run)
```

- [ ] **Step 4: Add confirmUpload method**

Add the following method to `service.go` (near `runUploadStub`):

```go
// confirmUpload queues all selected clips for the run to the upload_queue directory,
// inserts uploads rows, then advances the run to DONE.
func (s *Service) confirmUpload(ctx context.Context, id int64, run *database.PipelineRun) (AdvanceResult, error) {
	jobKey := fmt.Sprintf("%d", id)

	if !run.ChannelID.Valid {
		return AdvanceResult{}, fmt.Errorf("run %d has no channel_id set", id)
	}
	channelID := run.ChannelID.Int64

	// Load channel config for metadata
	cfg, err := s.channelCfgRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return AdvanceResult{}, fmt.Errorf("load channel config for channel %d: %w", channelID, err)
	}

	// Load selected clips
	allClips, err := s.clipRepo.ListByRun(ctx, id)
	if err != nil {
		return AdvanceResult{}, fmt.Errorf("load clips for run %d: %w", id, err)
	}
	var selected []database.PipelineClip
	for _, c := range allClips {
		if c.IsSelected {
			selected = append(selected, c)
		}
	}
	if len(selected) == 0 {
		return AdvanceResult{}, fmt.Errorf("no selected clips to queue for run %d", id)
	}

	// Queue each clip
	for i, clip := range selected {
		meta := uploader.BuildMetadata(clip, *cfg)
		metaBytes, _ := json.Marshal(meta)

		qVideo, qThumb, saveErr := s.queueStorage.SaveToQueue(clip.FilePath, clip.ThumbnailPath, meta)
		if saveErr != nil {
			s.log.Error("save to queue failed", "clip_id", clip.ID, "err", saveErr)
			return AdvanceResult{}, fmt.Errorf("save clip %d to queue: %w", clip.ID, saveErr)
		}

		_, createErr := s.uploadRepo.CreateQueued(ctx, channelID, clip.ID, qVideo, qThumb, string(metaBytes), "{}", i+1)
		if createErr != nil {
			s.log.Error("create queued upload failed", "clip_id", clip.ID, "err", createErr)
			return AdvanceResult{}, fmt.Errorf("create upload record for clip %d: %w", clip.ID, createErr)
		}
	}

	// Advance run to DONE
	if err := s.repo.AdvanceState(ctx, id, StateWaitingUploadConfirm, StateDone); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
				return AdvanceResult{State: curr.State}, nil
			}
		}
		return AdvanceResult{}, fmt.Errorf("advance run %d to done: %w", id, err)
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: StateDone},
	})
	s.log.Info("upload confirm: clips queued", "run_id", id, "count", len(selected))
	return AdvanceResult{State: StateDone}, nil
}
```

- [ ] **Step 5: Build check (will fail — main.go needs updating)**

Run: `cd server && go build ./...`
Expected: compile error in `cmd/server/main.go` — wrong number of arguments to `NewService`.

Proceed to Task 6 before expecting a clean build.

---

## Task 6: Wire main.go — new service deps

**Files:**
- Modify: `server/cmd/server/main.go`

- [ ] **Step 1: Add uploadRepo + queueStorage init in main.go**

Find the `pipeline.NewService` call (around line 85). Add the new repos before the call:

```go
uploadRepo := database.NewUploadRepo(db, slog.Default())
queueStorage := uploader.NewQueueStorage(*dir)
```

Update the `pipeline.NewService` call to:

```go
pipelineSvc := pipeline.NewService(repo, historyRepo, highlightRepo, clipRepo, channelCfgRepo, sseHub, ytDl, twDl, *dir, settingRepo, uploadRepo, queueStorage)
```

Add the `uploader` import:

```go
"github.com/joaoGMPereira/autocut/server/internal/uploader"
```

- [ ] **Step 2: Build check**

Run: `cd server && go build ./...`
Expected: success

- [ ] **Step 3: Commit Tasks 5+6 together**

```bash
git add server/internal/pipeline/service.go server/cmd/server/main.go
git commit -m "feat(pipeline): implement confirmUpload — queues selected clips, writes upload_queue files, MadeForKids=false"
```

---

## Task 7: Implement QueueHandler

**Files:**
- Modify: `server/internal/api/handlers/queue_handler.go`

- [ ] **Step 1: Write handler test**

Create `server/internal/api/handlers/queue_handler_test.go`:

```go
package handlers_test

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetQueue_ReturnsEmptyArray(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	db.Exec(`CREATE TABLE uploads (
		id INTEGER PRIMARY KEY, clip_id INTEGER, channel_id INTEGER,
		status TEXT, local_video_path TEXT, queue_order INTEGER,
		publish_at TEXT, youtube_id TEXT, youtube_url TEXT,
		error TEXT, created_at INTEGER
	)`)
	db.Exec(`CREATE TABLE pipeline_clips (id INTEGER PRIMARY KEY, title TEXT, thumbnail_path TEXT, duration_sec REAL)`)

	repo := database.NewUploadRepo(db, slog.Default())
	h := NewQueueHandler(repo, "/tmp")

	req := httptest.NewRequest(http.MethodGet, "/api/queue", nil)
	rr := httptest.NewRecorder()
	h.GetQueue(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var items []interface{}
	if err := json.NewDecoder(rr.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if items == nil {
		t.Fatal("expected [] not null")
	}
}
```

Run: `cd server && go test ./internal/api/handlers/... -run TestGetQueue`
Expected: FAIL — `NewQueueHandler` undefined.

- [ ] **Step 2: Implement queue_handler.go**

```go
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// QueueHandler serves the /api/queue endpoints.
type QueueHandler struct {
	repo    *database.UploadRepo
	dataDir string
	log     *slog.Logger
}

func NewQueueHandler(repo *database.UploadRepo, dataDir string) *QueueHandler {
	return &QueueHandler{repo: repo, dataDir: dataDir, log: slog.With("handler", "queue")}
}

// queueItemJSON is the JSON shape returned by GET /api/queue.
type queueItemJSON struct {
	ID            int64   `json:"id"`
	ClipID        int64   `json:"clip_id"`
	ChannelID     int64   `json:"channel_id"`
	Status        string  `json:"status"`
	VideoPath     string  `json:"video_path"`
	QueueOrder    int     `json:"queue_order"`
	PublishAt     *string `json:"publish_at,omitempty"`
	YoutubeID     string  `json:"youtube_id"`
	YoutubeURL    string  `json:"youtube_url"`
	Error         string  `json:"error"`
	CreatedAt     int64   `json:"created_at"`
	Title         string  `json:"title"`
	ThumbnailPath string  `json:"thumbnail_path"`
	VideoType     string  `json:"video_type"`
}

// GET /api/queue
func (h *QueueHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListQueueWithTitle(r.Context())
	if err != nil {
		h.log.Error("list queue", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := make([]queueItemJSON, 0, len(items))
	for _, q := range items {
		j := queueItemJSON{
			ID:            q.ID,
			ClipID:        q.ClipID,
			ChannelID:     q.ChannelID,
			Status:        q.Status,
			VideoPath:     q.VideoPath,
			QueueOrder:    q.QueueOrder,
			YoutubeID:     q.YoutubeID,
			YoutubeURL:    q.YoutubeURL,
			Error:         q.Error,
			CreatedAt:     q.CreatedAt,
			Title:         q.Title,
			ThumbnailPath: q.ThumbnailPath,
			VideoType:     q.VideoType,
		}
		if q.PublishAt.Valid {
			j.PublishAt = &q.PublishAt.String
		}
		out = append(out, j)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// DELETE /api/queue/{id}
func (h *QueueHandler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.log.Error("delete queue item", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/{id}/retry
func (h *QueueHandler) PostRetry(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateStatus(r.Context(), id, "queued"); err != nil {
		h.log.Error("retry queue item", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/{id}/schedule  body: {"publish_at":"2026-04-22T10:00:00Z"}
func (h *QueueHandler) PostSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		PublishAt string `json:"publish_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PublishAt == "" {
		http.Error(w, "publish_at required", http.StatusBadRequest)
		return
	}
	// Load current upload_config_json and set scheduled privacy
	u, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	configJSON := u.UploadConfigJSON
	if configJSON == "" {
		configJSON = "{}"
	}
	if err := h.repo.SetSchedule(r.Context(), id, body.PublishAt, configJSON); err != nil {
		h.log.Error("set schedule", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/bulk-schedule  body: {"start_at":"...","interval_minutes":1440}
func (h *QueueHandler) PostBulkSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StartAt         string `json:"start_at"`
		IntervalMinutes int    `json:"interval_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.StartAt == "" {
		http.Error(w, "start_at required", http.StatusBadRequest)
		return
	}
	if body.IntervalMinutes <= 0 {
		body.IntervalMinutes = 1440
	}
	items, err := h.repo.ListQueueWithTitle(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Parse start_at and schedule each queued item
	t, err := parseTime(body.StartAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid start_at: %v", err), http.StatusBadRequest)
		return
	}
	for _, item := range items {
		if item.Status != "queued" {
			continue
		}
		publishAt := t.UTC().Format("2006-01-02T15:04:05Z")
		_ = h.repo.SetSchedule(r.Context(), item.ID, publishAt, "{}")
		t = t.Add(minutesDur(body.IntervalMinutes))
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/save-local — no-op (files are already saved during upload confirm)
func (h *QueueHandler) PostSaveLocal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// GET /api/queue/{id}/thumbnail — serves the thumbnail file for a queue item
func (h *QueueHandler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	u, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Prefer local_thumbnail_path, fall back to pipeline_clips.thumbnail_path via ListQueueWithTitle
	thumbPath := ""
	if u.LocalThumbnailPath.Valid && u.LocalThumbnailPath.String != "" {
		thumbPath = u.LocalThumbnailPath.String
	} else {
		// Fall back: look up clip's thumbnail_path via ListQueueWithTitle filter
		all, _ := h.repo.ListQueueWithTitle(r.Context())
		for _, q := range all {
			if q.ID == id && q.ThumbnailPath != "" {
				thumbPath = q.ThumbnailPath
				break
			}
		}
	}

	if thumbPath == "" {
		http.Error(w, "no thumbnail", http.StatusNotFound)
		return
	}
	if _, err := os.Stat(thumbPath); err != nil {
		http.Error(w, "thumbnail file not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, thumbPath)
}

// parseIDParam extracts a named path parameter as int64.
func parseIDParam(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

// parseTime parses an ISO 8601 time string.
func parseTime(s string) (interface{ Add(d interface{}) interface{} }, error) {
	return nil, fmt.Errorf("not implemented")
}

// minutesDur converts minutes to a duration (placeholder replaced below).
func minutesDur(m int) interface{} { return nil }
```

Wait — the `PostBulkSchedule` time helpers are incomplete. Replace the whole handler with the proper version:

Full `queue_handler.go`:

```go
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// QueueHandler serves the /api/queue endpoints.
type QueueHandler struct {
	repo    *database.UploadRepo
	dataDir string
	log     *slog.Logger
}

// NewQueueHandler constructs a QueueHandler.
func NewQueueHandler(repo *database.UploadRepo, dataDir string) *QueueHandler {
	return &QueueHandler{repo: repo, dataDir: dataDir, log: slog.With("handler", "queue")}
}

// queueItemJSON is the JSON shape returned by GET /api/queue.
type queueItemJSON struct {
	ID            int64   `json:"id"`
	ClipID        int64   `json:"clip_id"`
	ChannelID     int64   `json:"channel_id"`
	Status        string  `json:"status"`
	VideoPath     string  `json:"video_path"`
	QueueOrder    int     `json:"queue_order"`
	PublishAt     *string `json:"publish_at,omitempty"`
	YoutubeID     string  `json:"youtube_id"`
	YoutubeURL    string  `json:"youtube_url"`
	Error         string  `json:"error"`
	CreatedAt     int64   `json:"created_at"`
	Title         string  `json:"title"`
	ThumbnailPath string  `json:"thumbnail_path"`
	VideoType     string  `json:"video_type"`
}

// GET /api/queue
func (h *QueueHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListQueueWithTitle(r.Context())
	if err != nil {
		h.log.Error("list queue", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := make([]queueItemJSON, 0, len(items))
	for _, q := range items {
		j := queueItemJSON{
			ID:            q.ID,
			ClipID:        q.ClipID,
			ChannelID:     q.ChannelID,
			Status:        q.Status,
			VideoPath:     q.VideoPath,
			QueueOrder:    q.QueueOrder,
			YoutubeID:     q.YoutubeID,
			YoutubeURL:    q.YoutubeURL,
			Error:         q.Error,
			CreatedAt:     q.CreatedAt,
			Title:         q.Title,
			ThumbnailPath: q.ThumbnailPath,
			VideoType:     q.VideoType,
		}
		if q.PublishAt.Valid {
			j.PublishAt = &q.PublishAt.String
		}
		out = append(out, j)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// DELETE /api/queue/{id}
func (h *QueueHandler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.log.Error("delete queue item", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/{id}/retry
func (h *QueueHandler) PostRetry(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateStatus(r.Context(), id, "queued"); err != nil {
		h.log.Error("retry queue item", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/{id}/schedule  body: {"publish_at":"2026-04-22T10:00:00Z"}
func (h *QueueHandler) PostSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		PublishAt string `json:"publish_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PublishAt == "" {
		http.Error(w, "publish_at required", http.StatusBadRequest)
		return
	}
	u, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	configJSON := u.UploadConfigJSON
	if configJSON == "" {
		configJSON = "{}"
	}
	if err := h.repo.SetSchedule(r.Context(), id, body.PublishAt, configJSON); err != nil {
		h.log.Error("set schedule", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/bulk-schedule  body: {"start_at":"2026-04-22T10:00:00Z","interval_minutes":1440}
func (h *QueueHandler) PostBulkSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StartAt         string `json:"start_at"`
		IntervalMinutes int    `json:"interval_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.StartAt == "" {
		http.Error(w, "start_at required", http.StatusBadRequest)
		return
	}
	if body.IntervalMinutes <= 0 {
		body.IntervalMinutes = 1440
	}
	t, err := time.Parse(time.RFC3339, body.StartAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid start_at: %v", err), http.StatusBadRequest)
		return
	}
	items, err := h.repo.ListQueueWithTitle(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for _, item := range items {
		if item.Status != "queued" {
			continue
		}
		publishAt := t.UTC().Format(time.RFC3339)
		_ = h.repo.SetSchedule(r.Context(), item.ID, publishAt, "{}")
		t = t.Add(time.Duration(body.IntervalMinutes) * time.Minute)
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/queue/save-local — no-op (files are saved during upload confirm)
func (h *QueueHandler) PostSaveLocal(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// GET /api/queue/{id}/thumbnail — serves the thumbnail file
func (h *QueueHandler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	id, err := queueParseID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	u, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	thumbPath := ""
	if u.LocalThumbnailPath.Valid && u.LocalThumbnailPath.String != "" {
		thumbPath = u.LocalThumbnailPath.String
	} else {
		all, _ := h.repo.ListQueueWithTitle(r.Context())
		for _, q := range all {
			if q.ID == id && q.ThumbnailPath != "" {
				thumbPath = q.ThumbnailPath
				break
			}
		}
	}

	if thumbPath == "" {
		http.Error(w, "no thumbnail", http.StatusNotFound)
		return
	}
	if _, statErr := os.Stat(thumbPath); statErr != nil {
		http.Error(w, "thumbnail file not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, thumbPath)
}

func queueParseID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
```

- [ ] **Step 3: Run test**

Run: `cd server && go test ./internal/api/handlers/... -run TestGetQueue`
Expected: PASS

- [ ] **Step 4: Build check**

Run: `cd server && go build ./...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add server/internal/api/handlers/queue_handler.go server/internal/api/handlers/queue_handler_test.go
git commit -m "feat(queue): implement QueueHandler with CRUD, bulk-schedule, thumbnail endpoint"
```

---

## Task 8: Wire queue routes in router.go + main.go

**Files:**
- Modify: `server/internal/api/router.go`
- Modify: `server/cmd/server/main.go`

- [ ] **Step 1: Add QueueHandler to NewRouter signature**

In `router.go`, update `NewRouter` signature to add `qh *handlers.QueueHandler` as the last parameter, and add queue routes:

```go
func NewRouter(
	ph *handlers.PipelineHandler,
	dh *handlers.DownloadHandler,
	sh *handlers.SetupHandler,
	sth *handlers.StatsHandler,
	uhh *handlers.URLHistoryHandler,
	seth *handlers.SettingsHandler,
	ch *handlers.ChannelHandler,
	oh *handlers.OAuthHandler,
	olh *handlers.OllamaHandler,
	mlh *handlers.MediaLibraryHandler,
	pvh *handlers.PreviewHandler,
	th *handlers.ThumbnailHandler,
	mh *handlers.MetadataHandler,
	qh *handlers.QueueHandler,      // NEW
) http.Handler {
```

Add queue routes after the existing Metadata endpoints block:

```go
// Queue endpoints
mux.HandleFunc("GET /api/queue", qh.GetQueue)
mux.HandleFunc("DELETE /api/queue/{id}", qh.DeleteQueue)
mux.HandleFunc("POST /api/queue/{id}/retry", qh.PostRetry)
mux.HandleFunc("POST /api/queue/{id}/schedule", qh.PostSchedule)
mux.HandleFunc("POST /api/queue/bulk-schedule", qh.PostBulkSchedule)
mux.HandleFunc("POST /api/queue/save-local", qh.PostSaveLocal)
mux.HandleFunc("GET /api/queue/{id}/thumbnail", qh.GetThumbnail)
```

- [ ] **Step 2: Initialize QueueHandler in main.go and pass to NewRouter**

In `main.go`, add after the `thumbnailHandler` line:

```go
queueHandler := handlers.NewQueueHandler(uploadRepo, *dir)
```

Update the `api.NewRouter(...)` call to pass `queueHandler` as the last argument.

- [ ] **Step 3: Build check**

Run: `cd server && go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add server/internal/api/router.go server/cmd/server/main.go
git commit -m "feat(router): register all /api/queue routes"
```

---

## Task 9: StepReviewMetadata — ThumbnailTypeBadge

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepReviewMetadata.tsx`

- [ ] **Step 1: Add ThumbnailTypeBadge component and wire it into card header**

In `StepReviewMetadata.tsx`, add `ThumbnailTypeBadge` after the closing `StepRow` helper at the bottom of the file:

```tsx
function ThumbnailTypeBadge({ style }: { style: string }) {
  const isLandscape = style === 'landscape';
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-semibold ${
        isLandscape
          ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
          : 'bg-violet-500/15 text-violet-400 border-violet-500/30'
      }`}
    >
      {isLandscape ? 'Landscape' : 'Shorts'}
    </span>
  );
}
```

In the clip card header row (around line 271–278), insert `<ThumbnailTypeBadge>` between the `<Badge>Clip {idx + 1}</Badge>` and the duration `<span>`:

Replace:
```tsx
<div className="flex items-center gap-2">
  <Badge variant="secondary" className="text-xs">
    Clip {idx + 1}
  </Badge>
  <span className="text-xs text-zinc-500">
    {formatDuration(clip.start_sec)} – {formatDuration(clip.end_sec)} ({formatDuration(clip.duration_sec)})
  </span>
</div>
```

With:
```tsx
<div className="flex items-center gap-2">
  <Badge variant="secondary" className="text-xs">
    Clip {idx + 1}
  </Badge>
  <ThumbnailTypeBadge style={clip.thumbnail_style} />
  <span className="text-xs text-zinc-500">
    {formatDuration(clip.start_sec)} – {formatDuration(clip.end_sec)} ({formatDuration(clip.duration_sec)})
  </span>
</div>
```

- [ ] **Step 2: TypeScript check**

Run: `cd apps/web && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add apps/web/src/components/pipeline/steps/StepReviewMetadata.tsx
git commit -m "feat(ui): add ThumbnailTypeBadge (Shorts/Landscape) to StepReviewMetadata clip cards"
```

---

## Task 10: Update queueStore — add thumbnail_path to QueueItem

**Files:**
- Modify: `apps/web/src/store/queueStore.ts`

- [ ] **Step 1: Add thumbnail_path field to QueueItem interface**

In `queueStore.ts`, find the `QueueItem` interface and add `thumbnail_path`:

```ts
export interface QueueItem {
  id: number;
  clip_id: number;
  channel_id: number;
  status: 'queued' | 'running' | 'done' | 'failed';
  youtube_id?: string;
  youtube_url?: string;
  queue_order: number;
  publish_at?: string;
  error?: string;
  title?: string;
  video_type: string;
  created_at: number;
  thumbnail_path?: string;  // NEW — present when clip has a thumbnail
}
```

- [ ] **Step 2: TypeScript check**

Run: `cd apps/web && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add apps/web/src/store/queueStore.ts
git commit -m "feat(queueStore): add thumbnail_path to QueueItem type"
```

---

## Task 11: Enrich QueueItemCard — thumbnail + channel badge

**Files:**
- Modify: `apps/web/src/components/queue/QueueItemCard.tsx`

- [ ] **Step 1: Update Props to include channelName and goUrl**

In `QueueItemCard.tsx`, update the `Props` interface and function signature:

```tsx
interface Props {
  item: QueueItem;
  channelName?: string;
  goUrl: string;
}

export function QueueItemCard({ item, channelName, goUrl }: Props) {
```

- [ ] **Step 2: Add thumbnail + channel badge to card layout**

Replace the current card `<div>` (the outer flex container starting around line 75) with:

```tsx
return (
  <div className="rounded-xl bg-card border border-border p-4 flex gap-3 items-start">
    {/* Thumbnail */}
    <div className="shrink-0 w-20 rounded-lg overflow-hidden bg-zinc-800 aspect-video flex items-center justify-center">
      {item.thumbnail_path ? (
        <img
          src={`${goUrl}/api/queue/${item.id}/thumbnail`}
          alt="thumbnail"
          className="w-full h-full object-cover"
        />
      ) : (
        <div className="w-full h-full bg-zinc-700" />
      )}
    </div>

    {/* Main content */}
    <div className="flex-1 min-w-0 flex flex-col gap-2">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-sm font-semibold text-[#F0F0F8] truncate">
          {item.title ?? 'Untitled'}
        </span>
        <StatusBadge status={item.status} />
        <span className="text-[11px] text-[#5C5C80] font-mono">{item.video_type}</span>
      </div>

      {channelName && (
        <span className="inline-flex w-fit items-center rounded-full border border-zinc-700 px-2 py-0.5 text-[10px] text-zinc-500">
          {channelName}
        </span>
      )}

      {item.publish_at && (
        <span className="inline-flex w-fit items-center gap-1 rounded-full bg-violet-500/15 border border-violet-500/30 px-2 py-0.5 text-[11px] font-medium text-violet-400">
          <Calendar className="size-3" />
          Scheduled: {new Date(item.publish_at).toLocaleString()}
        </span>
      )}

      {item.status === 'failed' && item.error && (
        <p className="text-xs text-red-400 bg-red-500/10 rounded-md px-3 py-1.5 border border-red-500/20">
          {item.error}
        </p>
      )}

      {item.status === 'done' && item.youtube_url && (
        <a
          href={item.youtube_url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-xs text-[#00D4FF] hover:underline w-fit"
        >
          <ExternalLink className="size-3" />
          View on YouTube
        </a>
      )}

      {/* Inline schedule picker */}
      {scheduleOpen && (
        <div className="flex items-center gap-2 mt-1">
          <input
            type="datetime-local"
            value={scheduleDraft}
            onChange={(e) => setScheduleDraft(e.target.value)}
            className="rounded-lg bg-[#1A1A26] border border-border px-2.5 py-1.5 text-xs text-[#F0F0F8] focus:outline-none focus:border-[#00D4FF]/60"
          />
          <Button
            size="sm"
            className="h-7 text-xs"
            disabled={!scheduleDraft || actionLoading === 'schedule'}
            onClick={() => void handleSchedule()}
          >
            {actionLoading === 'schedule' ? 'Saving…' : 'Set'}
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-7 text-xs"
            onClick={() => { setScheduleOpen(false); setScheduleDraft(''); }}
          >
            Cancel
          </Button>
        </div>
      )}
    </div>

    {/* Action buttons */}
    <TooltipProvider>
      <div className="flex items-center gap-1.5 shrink-0">
        {item.status === 'failed' && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-7 w-7 p-0"
                disabled={actionLoading === 'retry'}
                onClick={() => void handleRetry()}
              >
                <RotateCcw className="size-3.5" />
                <span className="sr-only">Retry</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Retry</TooltipContent>
          </Tooltip>
        )}

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className="h-7 w-7 p-0"
              onClick={() => setScheduleOpen((v) => !v)}
            >
              <Calendar className="size-3.5" />
              <span className="sr-only">Schedule</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>Schedule publish time</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className="h-7 w-7 p-0 text-red-400 hover:text-red-300 hover:border-red-500/50"
              disabled={actionLoading === 'delete'}
              onClick={() => void handleDelete()}
            >
              <Trash2 className="size-3.5" />
              <span className="sr-only">Delete</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>Remove from queue</TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  </div>
);
```

- [ ] **Step 3: TypeScript check**

Run: `cd apps/web && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/components/queue/QueueItemCard.tsx
git commit -m "feat(ui): enrich QueueItemCard with thumbnail image and channel badge"
```

---

## Task 12: Implement QueuePage with channel filter

**Files:**
- Modify: `apps/web/src/app/queue/page.tsx`

- [ ] **Step 1: Replace NotImplemented with real QueuePage**

Replace the entire file contents with:

```tsx
'use client';

import { useEffect, useState } from 'react';
import { useQueueStore } from '@/store/queueStore';
import { useChannelStore } from '@/store/channelStore';
import { useAppStore } from '@/store/appStore';
import { QueueItemCard } from '@/components/queue/QueueItemCard';
import { Button } from '@/components/ui/button';
import { createLogger } from '@/lib/logger';

const log = createLogger('QueuePage');

export default function QueuePage() {
  const goUrl = useAppStore((s) => s.goUrl);
  const { items, loading, error, fetchQueue, bulkSchedule } = useQueueStore();
  const { channels, fetchChannels } = useChannelStore();
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [bulkStartAt, setBulkStartAt] = useState('');
  const [bulkInterval, setBulkInterval] = useState(1440);
  const [bulkLoading, setBulkLoading] = useState(false);

  useEffect(() => {
    void fetchQueue();
    if (channels.length === 0) void fetchChannels(goUrl);
  }, [goUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  // Unique channel IDs that appear in the queue
  const channelIdsInQueue = [...new Set(items.map((i) => i.channel_id))];
  const channelOptions = channelIdsInQueue.map((cid) => ({
    id: cid,
    name: channels.find((c) => c.ID === cid)?.ChannelTitle ?? `Channel ${cid}`,
  }));

  const visibleItems = selectedChannelId
    ? items.filter((i) => i.channel_id === selectedChannelId)
    : items;

  const handleBulkSchedule = async () => {
    if (!bulkStartAt) return;
    setBulkLoading(true);
    try {
      await bulkSchedule(new Date(bulkStartAt).toISOString(), bulkInterval);
      setBulkOpen(false);
      setBulkStartAt('');
    } catch (err) {
      log.error('bulk schedule failed', { err });
    } finally {
      setBulkLoading(false);
    }
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Upload Queue</h1>
          <p className="text-sm text-zinc-500 mt-1">
            {items.length} item{items.length !== 1 ? 's' : ''} in queue
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => setBulkOpen((v) => !v)}>
          Bulk Schedule
        </Button>
      </div>

      {/* Bulk schedule form */}
      {bulkOpen && (
        <div className="rounded-xl border border-border bg-card p-4 flex flex-wrap items-end gap-3">
          <div className="flex flex-col gap-1">
            <label className="text-xs text-zinc-400">Start at</label>
            <input
              type="datetime-local"
              value={bulkStartAt}
              onChange={(e) => setBulkStartAt(e.target.value)}
              className="rounded-lg bg-[#1A1A26] border border-border px-2.5 py-1.5 text-xs text-[#F0F0F8] focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-zinc-400">Interval (min)</label>
            <input
              type="number"
              min={1}
              value={bulkInterval}
              onChange={(e) => setBulkInterval(Number(e.target.value))}
              className="w-24 rounded-lg bg-[#1A1A26] border border-border px-2.5 py-1.5 text-xs text-[#F0F0F8] focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
          <Button
            size="sm"
            disabled={!bulkStartAt || bulkLoading}
            onClick={() => void handleBulkSchedule()}
          >
            {bulkLoading ? 'Scheduling…' : 'Apply'}
          </Button>
          <Button size="sm" variant="outline" onClick={() => setBulkOpen(false)}>
            Cancel
          </Button>
        </div>
      )}

      {/* Channel filter pills */}
      {channelOptions.length > 1 && (
        <div className="flex items-center gap-2 flex-wrap">
          <button
            onClick={() => setSelectedChannelId(null)}
            className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
              selectedChannelId === null
                ? 'bg-primary text-primary-foreground border-primary'
                : 'border-border text-zinc-400 hover:border-zinc-500'
            }`}
          >
            Todos
          </button>
          {channelOptions.map((c) => (
            <button
              key={c.id}
              onClick={() => setSelectedChannelId(c.id === selectedChannelId ? null : c.id)}
              className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
                selectedChannelId === c.id
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'border-border text-zinc-400 hover:border-zinc-500'
              }`}
            >
              {c.name}
            </button>
          ))}
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="rounded-md border border-red-800 bg-red-950 px-4 py-3 text-xs text-red-400">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && items.length === 0 && (
        <p className="text-sm text-zinc-500">Carregando fila...</p>
      )}

      {/* Empty state */}
      {!loading && visibleItems.length === 0 && (
        <div className="rounded-xl border border-dashed border-zinc-700 p-10 text-center">
          <p className="text-sm text-zinc-500">Nenhum item na fila.</p>
          <p className="text-xs text-zinc-600 mt-1">
            Confirme um upload no pipeline para adicionar clips aqui.
          </p>
        </div>
      )}

      {/* Queue items */}
      <div className="space-y-3">
        {visibleItems.map((item) => {
          const channelName = channels.find((c) => c.ID === item.channel_id)?.ChannelTitle;
          return (
            <QueueItemCard
              key={item.id}
              item={item}
              channelName={channelName}
              goUrl={goUrl}
            />
          );
        })}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: TypeScript check**

Run: `cd apps/web && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add apps/web/src/app/queue/page.tsx
git commit -m "feat(ui): implement QueuePage with channel filter pills, thumbnail cards, bulk schedule"
```

---

## Self-Review Checklist

After all tasks complete, verify:

1. `cd server && go build ./...` — clean
2. `cd server && go test ./internal/uploader/... ./internal/database/... ./internal/api/handlers/... -v` — all pass
3. `cd apps/web && npx tsc --noEmit` — no type errors
4. Run the server locally and verify:
   - Pipeline run → review metadata screen shows Landscape/Shorts badge on each clip
   - Pipeline run → upload confirm → check `~/.autocut-dev/upload_queue/` contains `{ts}_{rnd}/video.mp4`, `thumbnail.jpg`, `metadata.json`
   - Check `metadata.json` has `"made_for_kids": false`
   - Queue page shows items with thumbnails and channel badges
   - Channel filter pills appear when multiple channels are present
