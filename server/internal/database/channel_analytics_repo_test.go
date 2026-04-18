package database

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// openTestDB opens an in-memory SQLite database and applies all migrations.
// The caller is responsible for closing the returned *sql.DB.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	// Foreign keys must be enabled before migrations run.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		t.Fatalf("enable foreign_keys: %v", err)
	}
	if err := migrate(db, slog.Default()); err != nil {
		db.Close()
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

// insertTestChannel inserts a minimal channel row and returns its id.
func insertTestChannel(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO channels (name, channel_id) VALUES (?, ?)`,
		"test_channel", "UC_test",
	)
	if err != nil {
		t.Fatalf("insert test channel: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func TestChannelAnalyticsRepo_UpsertAndGet(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	channelID := insertTestChannel(t, db)
	repo := NewChannelAnalyticsRepo(db)
	ctx := context.Background()

	now := time.Now().Unix()
	a := &ChannelAnalytics{
		ChannelID:     channelID,
		FetchedAt:     now,
		VideoCount:    42,
		AvgViews:      1000,
		TopTitleWords: `[{"word":"go","count":5}]`,
		TopTags:       `[{"tag":"coding","count":3}]`,
		SuccessTitles: `["Great title"]`,
		RawTitles:     `["Great title","Another one"]`,
	}

	// Upsert on empty table → row inserted.
	if err := repo.Upsert(ctx, a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Get should return the row (TTL = 1 hour, fetched_at is now).
	got, err := repo.Get(ctx, channelID, 3600)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil, want non-nil")
	}
	if got.VideoCount != 42 {
		t.Errorf("VideoCount: got %d, want 42", got.VideoCount)
	}
	if got.AvgViews != 1000 {
		t.Errorf("AvgViews: got %d, want 1000", got.AvgViews)
	}
	if got.TopTitleWords != a.TopTitleWords {
		t.Errorf("TopTitleWords: got %q, want %q", got.TopTitleWords, a.TopTitleWords)
	}
	if got.ChannelID != channelID {
		t.Errorf("ChannelID: got %d, want %d", got.ChannelID, channelID)
	}
}

func TestChannelAnalyticsRepo_UpsertReplacesExisting(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	channelID := insertTestChannel(t, db)
	repo := NewChannelAnalyticsRepo(db)
	ctx := context.Background()

	now := time.Now().Unix()

	// First upsert.
	first := &ChannelAnalytics{
		ChannelID:     channelID,
		FetchedAt:     now,
		VideoCount:    10,
		AvgViews:      500,
		TopTitleWords: `[]`,
		TopTags:       `[]`,
		SuccessTitles: `[]`,
		RawTitles:     `[]`,
	}
	if err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Second upsert (same channel_id) with updated video_count.
	second := &ChannelAnalytics{
		ChannelID:     channelID,
		FetchedAt:     now,
		VideoCount:    99,
		AvgViews:      2000,
		TopTitleWords: `[]`,
		TopTags:       `[]`,
		SuccessTitles: `[]`,
		RawTitles:     `[]`,
	}
	if err := repo.Upsert(ctx, second); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	// Get should return the updated row.
	got, err := repo.Get(ctx, channelID, 3600)
	if err != nil {
		t.Fatalf("Get after second upsert: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil after second upsert")
	}
	if got.VideoCount != 99 {
		t.Errorf("VideoCount after replace: got %d, want 99", got.VideoCount)
	}
	if got.AvgViews != 2000 {
		t.Errorf("AvgViews after replace: got %d, want 2000", got.AvgViews)
	}
}

func TestChannelAnalyticsRepo_GetStaleRow(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	channelID := insertTestChannel(t, db)
	repo := NewChannelAnalyticsRepo(db)
	ctx := context.Background()

	// fetched_at is 90000 seconds ago — well beyond any reasonable TTL.
	stale := time.Now().Unix() - 90000
	a := &ChannelAnalytics{
		ChannelID:     channelID,
		FetchedAt:     stale,
		VideoCount:    5,
		AvgViews:      100,
		TopTitleWords: `[]`,
		TopTags:       `[]`,
		SuccessTitles: `[]`,
		RawTitles:     `[]`,
	}
	if err := repo.Upsert(ctx, a); err != nil {
		t.Fatalf("Upsert stale row: %v", err)
	}

	// TTL = 3600s, but fetched_at is 90000s ago → should return nil.
	got, err := repo.Get(ctx, channelID, 3600)
	if err != nil {
		t.Fatalf("Get stale: %v", err)
	}
	if got != nil {
		t.Errorf("Get stale: expected nil (cache miss), got %+v", got)
	}
}

func TestChannelAnalyticsRepo_GetNoRow(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	channelID := insertTestChannel(t, db)
	repo := NewChannelAnalyticsRepo(db)
	ctx := context.Background()

	// No upsert — table is empty.
	got, err := repo.Get(ctx, channelID, 3600)
	if err != nil {
		t.Fatalf("Get with no row: %v", err)
	}
	if got != nil {
		t.Errorf("Get with no row: expected nil, got %+v", got)
	}
}
