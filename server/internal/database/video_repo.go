package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// VideoRepo provides CRUD for videos.
// Kotlin ref: VideosTable in Tables.kt (queries inline in Kotlin codebase)
type VideoRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewVideoRepo(db *sql.DB, log *slog.Logger) *VideoRepo {
	return &VideoRepo{db: db, log: log.With("repo", "video")}
}

const videoCols = `id, channel_id, url, platform, title, file_path, duration, status, created_at`

func (r *VideoRepo) Create(ctx context.Context, v *Video) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO videos (channel_id, url, platform, title, file_path, duration, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ChannelID, v.URL, v.Platform, v.Title, v.FilePath, v.Duration, v.Status, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert video: %w", err)
	}
	return res.LastInsertId()
}

func (r *VideoRepo) GetByID(ctx context.Context, id int64) (*Video, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+videoCols+" FROM videos WHERE id = ?", id)
	return scanVideo(row)
}

func (r *VideoRepo) ListByChannel(ctx context.Context, channelID int64) ([]Video, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+videoCols+" FROM videos WHERE channel_id = ? ORDER BY created_at DESC", channelID)
	if err != nil {
		return nil, fmt.Errorf("list videos by channel: %w", err)
	}
	defer rows.Close()

	var result []Video
	for rows.Next() {
		v, err := scanVideoRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}

func (r *VideoRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE videos SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("update video status %d: %w", id, err)
	}
	return nil
}

func (r *VideoRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM videos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete video %d: %w", id, err)
	}
	return nil
}

func scanVideo(row *sql.Row) (*Video, error) {
	var v Video
	err := row.Scan(&v.ID, &v.ChannelID, &v.URL, &v.Platform, &v.Title, &v.FilePath, &v.Duration, &v.Status, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func scanVideoRows(rows *sql.Rows) (*Video, error) {
	var v Video
	err := rows.Scan(&v.ID, &v.ChannelID, &v.URL, &v.Platform, &v.Title, &v.FilePath, &v.Duration, &v.Status, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
