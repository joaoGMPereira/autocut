package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// RemoteVideo represents a cached YouTube video entry from the remote_videos table.
// FP-015: used for Mass Update — 1h TTL cache of the channel's video list.
type RemoteVideo struct {
	ID          int64
	ChannelID   int64
	YoutubeID   string
	Title       string
	Description string
	Tags        string // comma-separated
	CategoryID  int
	PublishedAt time.Time
	FetchedAt   time.Time
}

// RemoteVideoFilter applies optional date/category constraints to ListFiltered.
type RemoteVideoFilter struct {
	DateFrom   *time.Time
	DateTo     *time.Time
	CategoryID *int
}

// RemoteVideoRepo handles persistence for the remote_videos table.
type RemoteVideoRepo struct {
	db  *sql.DB
	log *slog.Logger
}

// NewRemoteVideoRepo creates a RemoteVideoRepo.
func NewRemoteVideoRepo(db *sql.DB, log *slog.Logger) *RemoteVideoRepo {
	return &RemoteVideoRepo{db: db, log: log.With("component", "db", "repo", "remote_video")}
}

// IsCacheFresh returns true if remote_videos for channelID has at least one row
// fetched within maxAge. Used to decide whether to hit the YouTube API again.
func (r *RemoteVideoRepo) IsCacheFresh(ctx context.Context, channelID int64, maxAge time.Duration) (bool, error) {
	threshold := time.Now().Add(-maxAge).UTC().Format(time.RFC3339)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM remote_videos WHERE channel_id = ? AND fetched_at >= ?`,
		channelID, threshold,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("cache freshness check: %w", err)
	}
	return count > 0, nil
}

// UpsertBatch inserts or replaces remote video records for a channel.
// On conflict (channel_id, youtube_id) it updates all mutable fields and refreshes fetched_at.
func (r *RemoteVideoRepo) UpsertBatch(ctx context.Context, channelID int64, videos []RemoteVideo) error {
	if len(videos) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO remote_videos (channel_id, youtube_id, title, description, tags, category_id, published_at, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_id, youtube_id) DO UPDATE SET
			title        = excluded.title,
			description  = excluded.description,
			tags         = excluded.tags,
			category_id  = excluded.category_id,
			published_at = excluded.published_at,
			fetched_at   = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("prepare upsert stmt: %w", err)
	}
	defer stmt.Close()

	for _, v := range videos {
		pubAt := ""
		if !v.PublishedAt.IsZero() {
			pubAt = v.PublishedAt.UTC().Format(time.RFC3339)
		}
		if _, err := stmt.ExecContext(ctx, channelID, v.YoutubeID, v.Title, v.Description, v.Tags, v.CategoryID, pubAt); err != nil {
			r.log.Error("upsert remote video failed", "err", err, "youtubeID", v.YoutubeID)
			return fmt.Errorf("upsert video %q: %w", v.YoutubeID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert tx: %w", err)
	}
	r.log.Info("remote videos upserted", "channelID", channelID, "count", len(videos))
	return nil
}

// ListByChannel returns a paginated list of remote videos for a channel, newest first.
func (r *RemoteVideoRepo) ListByChannel(ctx context.Context, channelID int64, page, pageSize int) ([]RemoteVideo, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM remote_videos WHERE channel_id = ?`, channelID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count remote videos: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, channel_id, youtube_id, title, description, tags, category_id, published_at, fetched_at
		FROM remote_videos
		WHERE channel_id = ?
		ORDER BY published_at DESC
		LIMIT ? OFFSET ?
	`, channelID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list remote videos: %w", err)
	}
	defer rows.Close()

	videos, err := scanRemoteVideos(rows)
	if err != nil {
		return nil, 0, err
	}
	return videos, total, nil
}

// ListFiltered returns remote videos for a channel matching the filter, up to limit rows.
// Used by the mass-update handler to select which videos to update.
func (r *RemoteVideoRepo) ListFiltered(ctx context.Context, channelID int64, f RemoteVideoFilter, limit int) ([]RemoteVideo, error) {
	q := `SELECT id, channel_id, youtube_id, title, description, tags, category_id, published_at, fetched_at
		  FROM remote_videos WHERE channel_id = ?`
	args := []any{channelID}

	if f.DateFrom != nil {
		q += " AND published_at >= ?"
		args = append(args, f.DateFrom.UTC().Format(time.RFC3339))
	}
	if f.DateTo != nil {
		q += " AND published_at <= ?"
		args = append(args, f.DateTo.UTC().Format(time.RFC3339))
	}
	if f.CategoryID != nil {
		q += " AND category_id = ?"
		args = append(args, *f.CategoryID)
	}
	q += " ORDER BY published_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list filtered remote videos: %w", err)
	}
	defer rows.Close()

	return scanRemoteVideos(rows)
}

// scanRemoteVideos reads all rows into a []RemoteVideo slice.
func scanRemoteVideos(rows *sql.Rows) ([]RemoteVideo, error) {
	var videos []RemoteVideo
	for rows.Next() {
		var v RemoteVideo
		var pubAt, fetchedAt sql.NullString
		if err := rows.Scan(
			&v.ID, &v.ChannelID, &v.YoutubeID,
			&v.Title, &v.Description, &v.Tags, &v.CategoryID,
			&pubAt, &fetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan remote video row: %w", err)
		}
		if pubAt.Valid && pubAt.String != "" {
			t, _ := time.Parse(time.RFC3339, pubAt.String)
			v.PublishedAt = t
		}
		if fetchedAt.Valid && fetchedAt.String != "" {
			t, _ := time.Parse(time.RFC3339, fetchedAt.String)
			v.FetchedAt = t
		}
		videos = append(videos, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	if videos == nil {
		videos = []RemoteVideo{}
	}
	return videos, nil
}
