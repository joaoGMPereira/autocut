package database_test

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"

	. "github.com/joaoGMPereira/autocut/server/internal/database"
	_ "modernc.org/sqlite"
)

func openQueueTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
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
			description TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
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
	db := openQueueTestDB(t)
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
	db := openQueueTestDB(t)
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
