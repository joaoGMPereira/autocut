package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// UploadRepo provides CRUD for uploads.
// Kotlin ref: UploadRepository.kt — the largest repo with 18+ methods.
type UploadRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewUploadRepo(db *sql.DB, log *slog.Logger) *UploadRepo {
	return &UploadRepo{db: db, log: log.With("repo", "upload")}
}

const uploadCols = `id, clip_id, channel_id, youtube_id, youtube_url, status,
	scheduled_at, uploaded_at, error, video_type,
	local_video_path, local_thumbnail_path, metadata_json, upload_config_json,
	original_video_name, source_video_url, source_clip_url,
	shorts_generated, shorts_generated_at, created_at`

func (r *UploadRepo) Create(ctx context.Context, u *Upload) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO uploads (clip_id, channel_id, youtube_id, youtube_url, status,
			scheduled_at, uploaded_at, error, video_type,
			local_video_path, local_thumbnail_path, metadata_json, upload_config_json,
			original_video_name, source_video_url, source_clip_url,
			shorts_generated, shorts_generated_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ClipID, u.ChannelID, u.YoutubeID, u.YoutubeURL, u.Status,
		u.ScheduledAt, u.UploadedAt, u.Error, u.VideoType,
		u.LocalVideoPath, u.LocalThumbnailPath, u.MetadataJSON, u.UploadConfigJSON,
		u.OriginalVideoName, u.SourceVideoURL, u.SourceClipURL,
		u.ShortsGenerated, u.ShortsGeneratedAt, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert upload: %w", err)
	}
	return res.LastInsertId()
}

// CreateLocal creates a local upload entry (saved_locally status).
// Kotlin ref: createLocalUpload(uploadPackage, scheduledAt)
func (r *UploadRepo) CreateLocal(ctx context.Context, u *Upload) (int64, error) {
	u.Status = "saved_locally"
	return r.Create(ctx, u)
}

func (r *UploadRepo) GetByID(ctx context.Context, id int64) (*Upload, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+uploadCols+" FROM uploads WHERE id = ?", id)
	return scanUpload(row)
}

// Kotlin ref: getAll(): List<Upload>
func (r *UploadRepo) List(ctx context.Context) ([]Upload, error) {
	return r.queryUploads(ctx, "SELECT "+uploadCols+" FROM uploads ORDER BY created_at DESC")
}

// Kotlin ref: getByStatus(status: String): List<Upload>
func (r *UploadRepo) ListByStatus(ctx context.Context, status string) ([]Upload, error) {
	return r.queryUploads(ctx, "SELECT "+uploadCols+" FROM uploads WHERE status = ? ORDER BY created_at DESC", status)
}

// Kotlin ref: combination of getByStatus + channel filter
func (r *UploadRepo) ListByChannelAndStatus(ctx context.Context, channelID int64, status string) ([]Upload, error) {
	return r.queryUploads(ctx,
		"SELECT "+uploadCols+" FROM uploads WHERE channel_id = ? AND status = ? ORDER BY created_at DESC",
		channelID, status)
}

func (r *UploadRepo) ListByChannel(ctx context.Context, channelID int64) ([]Upload, error) {
	return r.queryUploads(ctx,
		"SELECT "+uploadCols+" FROM uploads WHERE channel_id = ? ORDER BY created_at DESC", channelID)
}

func (r *UploadRepo) ListByClip(ctx context.Context, clipID int64) ([]Upload, error) {
	return r.queryUploads(ctx,
		"SELECT "+uploadCols+" FROM uploads WHERE clip_id = ? ORDER BY created_at DESC", clipID)
}

// Kotlin ref: getByVideoType(type: VideoType): List<Upload>
func (r *UploadRepo) ListByVideoType(ctx context.Context, videoType string) ([]Upload, error) {
	return r.queryUploads(ctx,
		"SELECT "+uploadCols+" FROM uploads WHERE video_type = ? ORDER BY created_at DESC", videoType)
}

// Kotlin ref: getUploadedToday(): List<Upload>
func (r *UploadRepo) ListUploadedToday(ctx context.Context) ([]Upload, error) {
	startOfDay := startOfDayMillis()
	return r.queryUploads(ctx,
		"SELECT "+uploadCols+" FROM uploads WHERE status = 'uploaded' AND uploaded_at >= ?", startOfDay)
}

// Kotlin ref: getUploadsWithoutShorts(channelId?): List<Upload>
func (r *UploadRepo) ListWithoutShorts(ctx context.Context, channelID sql.NullInt64) ([]Upload, error) {
	if channelID.Valid {
		return r.queryUploads(ctx,
			"SELECT "+uploadCols+" FROM uploads WHERE shorts_generated = 0 AND video_type = 'long_form' AND channel_id = ?",
			channelID.Int64)
	}
	return r.queryUploads(ctx,
		"SELECT "+uploadCols+" FROM uploads WHERE shorts_generated = 0 AND video_type = 'long_form'")
}

// Kotlin ref: updateStatus(id, status): Boolean
func (r *UploadRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE uploads SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("update upload status %d: %w", id, err)
	}
	return nil
}

func (r *UploadRepo) Update(ctx context.Context, u *Upload) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE uploads SET
			clip_id = ?, channel_id = ?, youtube_id = ?, youtube_url = ?, status = ?,
			scheduled_at = ?, uploaded_at = ?, error = ?, video_type = ?,
			local_video_path = ?, local_thumbnail_path = ?, metadata_json = ?, upload_config_json = ?,
			original_video_name = ?, source_video_url = ?, source_clip_url = ?,
			shorts_generated = ?, shorts_generated_at = ?
		 WHERE id = ?`,
		u.ClipID, u.ChannelID, u.YoutubeID, u.YoutubeURL, u.Status,
		u.ScheduledAt, u.UploadedAt, u.Error, u.VideoType,
		u.LocalVideoPath, u.LocalThumbnailPath, u.MetadataJSON, u.UploadConfigJSON,
		u.OriginalVideoName, u.SourceVideoURL, u.SourceClipURL,
		u.ShortsGenerated, u.ShortsGeneratedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update upload %d: %w", u.ID, err)
	}
	return nil
}

// Kotlin ref: markCompleted(id, youtubeId, youtubeUrl): Boolean
func (r *UploadRepo) MarkCompleted(ctx context.Context, id int64, youtubeID, youtubeURL string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE uploads SET status = 'uploaded', youtube_id = ?, youtube_url = ?, uploaded_at = ? WHERE id = ?",
		youtubeID, youtubeURL, now, id,
	)
	if err != nil {
		return fmt.Errorf("mark upload completed %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: markError(id, errorMessage): Boolean
func (r *UploadRepo) MarkError(ctx context.Context, id int64, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE uploads SET status = 'error', error = ? WHERE id = ?", errMsg, id)
	if err != nil {
		return fmt.Errorf("mark upload error %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: retry(id): Boolean — resets status to pending and clears error
func (r *UploadRepo) Retry(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE uploads SET status = 'pending', error = '' WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("retry upload %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: markShortsGenerated(uploadId): Boolean
func (r *UploadRepo) MarkShortsGenerated(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE uploads SET shorts_generated = 1, shorts_generated_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("mark shorts generated %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: countByStatus(): Map<String, Int>
func (r *UploadRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT status, COUNT(*) FROM uploads GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("count uploads by status: %w", err)
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

func (r *UploadRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM uploads WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete upload %d: %w", id, err)
	}
	return nil
}

// queryUploads is a helper that executes a query and scans upload rows.
func (r *UploadRepo) queryUploads(ctx context.Context, query string, args ...any) ([]Upload, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query uploads: %w", err)
	}
	defer rows.Close()

	var result []Upload
	for rows.Next() {
		u, err := scanUploadRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *u)
	}
	return result, rows.Err()
}

func scanUpload(row *sql.Row) (*Upload, error) {
	var u Upload
	err := row.Scan(
		&u.ID, &u.ClipID, &u.ChannelID, &u.YoutubeID, &u.YoutubeURL, &u.Status,
		&u.ScheduledAt, &u.UploadedAt, &u.Error, &u.VideoType,
		&u.LocalVideoPath, &u.LocalThumbnailPath, &u.MetadataJSON, &u.UploadConfigJSON,
		&u.OriginalVideoName, &u.SourceVideoURL, &u.SourceClipURL,
		&u.ShortsGenerated, &u.ShortsGeneratedAt, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func scanUploadRows(rows *sql.Rows) (*Upload, error) {
	var u Upload
	err := rows.Scan(
		&u.ID, &u.ClipID, &u.ChannelID, &u.YoutubeID, &u.YoutubeURL, &u.Status,
		&u.ScheduledAt, &u.UploadedAt, &u.Error, &u.VideoType,
		&u.LocalVideoPath, &u.LocalThumbnailPath, &u.MetadataJSON, &u.UploadConfigJSON,
		&u.OriginalVideoName, &u.SourceVideoURL, &u.SourceClipURL,
		&u.ShortsGenerated, &u.ShortsGeneratedAt, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// startOfDayMillis returns epoch millis for the start of today (UTC).
func startOfDayMillis() int64 {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return start.UnixMilli()
}
