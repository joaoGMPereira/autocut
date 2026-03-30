package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// ShortRepo provides CRUD for shorts.
// Kotlin ref: ShortsRepository.kt
type ShortRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewShortRepo(db *sql.DB, log *slog.Logger) *ShortRepo {
	return &ShortRepo{db: db, log: log.With("repo", "short")}
}

const shortCols = `id, source_video_path, segment_json, file_path,
	thumbnail_path, custom_thumbnail_path, title, thumbnail_title,
	description, tags, duration, channel_id,
	youtube_id, youtube_url, status, scheduled_at, uploaded_at, error, created_at`

func (r *ShortRepo) Create(ctx context.Context, s *Short) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO shorts (source_video_path, segment_json, file_path,
			thumbnail_path, custom_thumbnail_path, title, thumbnail_title,
			description, tags, duration, channel_id,
			youtube_id, youtube_url, status, scheduled_at, uploaded_at, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.SourceVideoPath, s.SegmentJSON, s.FilePath,
		s.ThumbnailPath, s.CustomThumbnailPath, s.Title, s.ThumbnailTitle,
		s.Description, s.Tags, s.Duration, s.ChannelID,
		s.YoutubeID, s.YoutubeURL, s.Status, s.ScheduledAt, s.UploadedAt, s.Error, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert short: %w", err)
	}
	return res.LastInsertId()
}

func (r *ShortRepo) GetByID(ctx context.Context, id int64) (*Short, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+shortCols+" FROM shorts WHERE id = ?", id)
	return scanShort(row)
}

// Kotlin ref: getAll(): List<ShortRecord>
func (r *ShortRepo) List(ctx context.Context) ([]Short, error) {
	return r.queryShorts(ctx, "SELECT "+shortCols+" FROM shorts ORDER BY created_at DESC")
}

// Kotlin ref: getByStatus(status): List<ShortRecord>
func (r *ShortRepo) ListByStatus(ctx context.Context, status string) ([]Short, error) {
	return r.queryShorts(ctx, "SELECT "+shortCols+" FROM shorts WHERE status = ? ORDER BY created_at DESC", status)
}

// Kotlin ref: getByChannelId(channelId): List<ShortRecord>
func (r *ShortRepo) ListByChannel(ctx context.Context, channelID int64) ([]Short, error) {
	return r.queryShorts(ctx, "SELECT "+shortCols+" FROM shorts WHERE channel_id = ? ORDER BY created_at DESC", channelID)
}

// Kotlin ref: updateMetadata(id, title?, thumbnailTitle?, description?, tags?)
func (r *ShortRepo) UpdateMetadata(ctx context.Context, id int64, title, thumbnailTitle, description, tags string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET title = ?, thumbnail_title = ?, description = ?, tags = ? WHERE id = ?",
		title, thumbnailTitle, description, tags, id,
	)
	if err != nil {
		return fmt.Errorf("update short metadata %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: setCustomThumbnail(id, thumbnailPath?)
func (r *ShortRepo) SetCustomThumbnail(ctx context.Context, id int64, path sql.NullString) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET custom_thumbnail_path = ? WHERE id = ?", path, id)
	if err != nil {
		return fmt.Errorf("set custom thumbnail %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: updateStatus(id, status)
func (r *ShortRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("update short status %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: setChannel(id, channelId)
func (r *ShortRepo) SetChannel(ctx context.Context, id int64, channelID int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET channel_id = ? WHERE id = ?", channelID, id)
	if err != nil {
		return fmt.Errorf("set short channel %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: markPending(id, channelId, scheduledAt?)
func (r *ShortRepo) MarkPending(ctx context.Context, id, channelID int64, scheduledAt sql.NullInt64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET status = 'pending', channel_id = ?, scheduled_at = ? WHERE id = ?",
		channelID, scheduledAt, id,
	)
	if err != nil {
		return fmt.Errorf("mark short pending %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: markCompleted(id, youtubeId, youtubeUrl)
func (r *ShortRepo) MarkCompleted(ctx context.Context, id int64, youtubeID, youtubeURL string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET status = 'uploaded', youtube_id = ?, youtube_url = ?, uploaded_at = ? WHERE id = ?",
		youtubeID, youtubeURL, now, id,
	)
	if err != nil {
		return fmt.Errorf("mark short completed %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: markError(id, errorMessage)
func (r *ShortRepo) MarkError(ctx context.Context, id int64, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET status = 'error', error = ? WHERE id = ?", errMsg, id)
	if err != nil {
		return fmt.Errorf("mark short error %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: retry(id) — resets status to created
func (r *ShortRepo) Retry(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE shorts SET status = 'created', error = '' WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("retry short %d: %w", id, err)
	}
	return nil
}

func (r *ShortRepo) Update(ctx context.Context, s *Short) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE shorts SET
			source_video_path = ?, segment_json = ?, file_path = ?,
			thumbnail_path = ?, custom_thumbnail_path = ?, title = ?, thumbnail_title = ?,
			description = ?, tags = ?, duration = ?, channel_id = ?,
			youtube_id = ?, youtube_url = ?, status = ?, scheduled_at = ?, uploaded_at = ?, error = ?
		 WHERE id = ?`,
		s.SourceVideoPath, s.SegmentJSON, s.FilePath,
		s.ThumbnailPath, s.CustomThumbnailPath, s.Title, s.ThumbnailTitle,
		s.Description, s.Tags, s.Duration, s.ChannelID,
		s.YoutubeID, s.YoutubeURL, s.Status, s.ScheduledAt, s.UploadedAt, s.Error, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update short %d: %w", s.ID, err)
	}
	return nil
}

// Kotlin ref: countByStatus(): Map<String, Int>
func (r *ShortRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT status, COUNT(*) FROM shorts GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("count shorts by status: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, rows.Err()
}

func (r *ShortRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM shorts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete short %d: %w", id, err)
	}
	return nil
}

func (r *ShortRepo) queryShorts(ctx context.Context, query string, args ...any) ([]Short, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query shorts: %w", err)
	}
	defer rows.Close()

	var result []Short
	for rows.Next() {
		s, err := scanShortRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}
	return result, rows.Err()
}

func scanShort(row *sql.Row) (*Short, error) {
	var s Short
	err := row.Scan(
		&s.ID, &s.SourceVideoPath, &s.SegmentJSON, &s.FilePath,
		&s.ThumbnailPath, &s.CustomThumbnailPath, &s.Title, &s.ThumbnailTitle,
		&s.Description, &s.Tags, &s.Duration, &s.ChannelID,
		&s.YoutubeID, &s.YoutubeURL, &s.Status, &s.ScheduledAt, &s.UploadedAt, &s.Error, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanShortRows(rows *sql.Rows) (*Short, error) {
	var s Short
	err := rows.Scan(
		&s.ID, &s.SourceVideoPath, &s.SegmentJSON, &s.FilePath,
		&s.ThumbnailPath, &s.CustomThumbnailPath, &s.Title, &s.ThumbnailTitle,
		&s.Description, &s.Tags, &s.Duration, &s.ChannelID,
		&s.YoutubeID, &s.YoutubeURL, &s.Status, &s.ScheduledAt, &s.UploadedAt, &s.Error, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
