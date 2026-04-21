package handlers_test

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/database"
)

func setupQueueHandlerTestDB(t *testing.T) *sql.DB {
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
			id INTEGER PRIMARY KEY,
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

func TestQueueHandler_GetQueue_Empty(t *testing.T) {
	db := setupQueueHandlerTestDB(t)
	repo := database.NewUploadRepo(db, slog.Default())
	h := handlers.NewQueueHandler(repo, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/queue", nil)
	rr := httptest.NewRecorder()
	h.GetQueue(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var items []interface{}
	if err := json.NewDecoder(rr.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty array, got %d items", len(items))
	}
}
