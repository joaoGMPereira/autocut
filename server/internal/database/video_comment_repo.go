package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// VideoComment represents a single YouTube top-level comment from video_comments table.
// FP-016: persisted after each comment sync job.
type VideoComment struct {
	ID             int64
	YoutubeVideoID string
	ChannelID      int64
	CommentID      string
	Author         string
	Text           string
	Likes          int64
	PublishedAt    time.Time
	SyncedAt       time.Time
}

// VideoCommentRepo handles persistence for the video_comments table.
type VideoCommentRepo struct {
	db  *sql.DB
	log *slog.Logger
}

// NewVideoCommentRepo creates a VideoCommentRepo.
func NewVideoCommentRepo(db *sql.DB, log *slog.Logger) *VideoCommentRepo {
	return &VideoCommentRepo{db: db, log: log.With("component", "db", "repo", "video_comment")}
}

// UpsertBatch inserts or updates comment records.
// ON CONFLICT(comment_id) updates mutable fields (text, likes, synced_at).
func (r *VideoCommentRepo) UpsertBatch(ctx context.Context, comments []VideoComment) error {
	if len(comments) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin comment upsert tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO video_comments (youtube_video_id, channel_id, comment_id, author, text, likes, published_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(comment_id) DO UPDATE SET
			author       = excluded.author,
			text         = excluded.text,
			likes        = excluded.likes,
			synced_at    = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("prepare comment upsert stmt: %w", err)
	}
	defer stmt.Close()

	for _, c := range comments {
		pubAt := ""
		if !c.PublishedAt.IsZero() {
			pubAt = c.PublishedAt.UTC().Format(time.RFC3339)
		}
		if _, err := stmt.ExecContext(ctx, c.YoutubeVideoID, c.ChannelID, c.CommentID, c.Author, c.Text, c.Likes, pubAt); err != nil {
			r.log.Error("upsert comment failed", "err", err, "commentID", c.CommentID)
			return fmt.Errorf("upsert comment %q: %w", c.CommentID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit comment upsert tx: %w", err)
	}
	r.log.Info("video comments upserted", "count", len(comments))
	return nil
}

// ListByChannel returns a paginated, optionally filtered list of comments for a channel.
// search matches against text or author (case-insensitive LIKE).
func (r *VideoCommentRepo) ListByChannel(ctx context.Context, channelID int64, page, pageSize int, search string) ([]VideoComment, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	whereClause := "WHERE channel_id = ?"
	args := []any{channelID}
	countArgs := []any{channelID}

	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		whereClause += " AND (LOWER(text) LIKE ? OR LOWER(author) LIKE ?)"
		args = append(args, pattern, pattern)
		countArgs = append(countArgs, pattern, pattern)
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM video_comments "+whereClause, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}

	q := `SELECT id, youtube_video_id, channel_id, comment_id, author, text, likes, published_at, synced_at
		  FROM video_comments ` + whereClause + ` ORDER BY published_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	var comments []VideoComment
	for rows.Next() {
		var c VideoComment
		var pubAt, syncedAt sql.NullString
		if err := rows.Scan(
			&c.ID, &c.YoutubeVideoID, &c.ChannelID, &c.CommentID,
			&c.Author, &c.Text, &c.Likes,
			&pubAt, &syncedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan comment row: %w", err)
		}
		if pubAt.Valid && pubAt.String != "" {
			t, _ := time.Parse(time.RFC3339, pubAt.String)
			c.PublishedAt = t
		}
		if syncedAt.Valid && syncedAt.String != "" {
			t, _ := time.Parse(time.RFC3339, syncedAt.String)
			c.SyncedAt = t
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}
	if comments == nil {
		comments = []VideoComment{}
	}
	return comments, total, nil
}
