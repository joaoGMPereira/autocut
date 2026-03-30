package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// ClipRepo provides CRUD for clips.
// Kotlin ref: ClipsTable in Tables.kt (queries inline in Kotlin codebase)
type ClipRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewClipRepo(db *sql.DB, log *slog.Logger) *ClipRepo {
	return &ClipRepo{db: db, log: log.With("repo", "clip")}
}

const clipCols = `id, video_id, start_time, end_time, file_path, title, description, tags, thumbnail_path, status, created_at`

func (r *ClipRepo) Create(ctx context.Context, c *Clip) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO clips (video_id, start_time, end_time, file_path, title, description, tags, thumbnail_path, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.VideoID, c.StartTime, c.EndTime, c.FilePath, c.Title, c.Description, c.Tags, c.ThumbnailPath, c.Status, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert clip: %w", err)
	}
	return res.LastInsertId()
}

func (r *ClipRepo) GetByID(ctx context.Context, id int64) (*Clip, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+clipCols+" FROM clips WHERE id = ?", id)
	return scanClip(row)
}

func (r *ClipRepo) ListByVideo(ctx context.Context, videoID int64) ([]Clip, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+clipCols+" FROM clips WHERE video_id = ? ORDER BY start_time ASC", videoID)
	if err != nil {
		return nil, fmt.Errorf("list clips by video: %w", err)
	}
	defer rows.Close()

	var result []Clip
	for rows.Next() {
		c, err := scanClipRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

func (r *ClipRepo) Update(ctx context.Context, c *Clip) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clips SET
			start_time = ?, end_time = ?, file_path = ?, title = ?,
			description = ?, tags = ?, thumbnail_path = ?, status = ?
		 WHERE id = ?`,
		c.StartTime, c.EndTime, c.FilePath, c.Title,
		c.Description, c.Tags, c.ThumbnailPath, c.Status, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update clip %d: %w", c.ID, err)
	}
	return nil
}

func (r *ClipRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE clips SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("update clip status %d: %w", id, err)
	}
	return nil
}

func (r *ClipRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM clips WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete clip %d: %w", id, err)
	}
	return nil
}

func scanClip(row *sql.Row) (*Clip, error) {
	var c Clip
	err := row.Scan(&c.ID, &c.VideoID, &c.StartTime, &c.EndTime, &c.FilePath, &c.Title, &c.Description, &c.Tags, &c.ThumbnailPath, &c.Status, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanClipRows(rows *sql.Rows) (*Clip, error) {
	var c Clip
	err := rows.Scan(&c.ID, &c.VideoID, &c.StartTime, &c.EndTime, &c.FilePath, &c.Title, &c.Description, &c.Tags, &c.ThumbnailPath, &c.Status, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
