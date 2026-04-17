package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// PipelineClipRepo provides CRUD for pipeline_clips (v9 schema).
type PipelineClipRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewPipelineClipRepo(db *sql.DB, log *slog.Logger) *PipelineClipRepo {
	return &PipelineClipRepo{db: db, log: log.With("repo", "pipeline_clip")}
}

const pipelineClipCols = `id, run_id, highlight_id, file_path, thumbnail_path, title, description, tags, thumbnail_style, thumbnail_text, is_selected, start_sec, end_sec, duration_sec, upload_status, youtube_id, created_at`

// Create inserts a new pipeline clip and returns its ID.
func (r *PipelineClipRepo) Create(ctx context.Context, c *PipelineClip) (int64, error) {
	now := time.Now().UnixMilli()
	uploadStatus := c.UploadStatus
	if uploadStatus == "" {
		uploadStatus = "pending"
	}
	thumbnailStyle := c.ThumbnailStyle
	if thumbnailStyle == "" {
		thumbnailStyle = "default"
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pipeline_clips (run_id, highlight_id, file_path, thumbnail_path, title, description, tags, thumbnail_style, thumbnail_text, is_selected, start_sec, end_sec, duration_sec, upload_status, youtube_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.RunID, c.HighlightID, c.FilePath, c.ThumbnailPath, c.Title, c.Description, c.Tags, thumbnailStyle, c.ThumbnailText, c.IsSelected, c.StartSec, c.EndSec, c.DurationSec, uploadStatus, c.YouTubeID, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert pipeline_clip: %w", err)
	}
	return res.LastInsertId()
}

// ListByRun returns all clips for a given run, ordered by start_sec.
func (r *PipelineClipRepo) ListByRun(ctx context.Context, runID int64) ([]PipelineClip, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+pipelineClipCols+" FROM pipeline_clips WHERE run_id = ? ORDER BY start_sec ASC",
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipeline_clips for run %d: %w", runID, err)
	}
	defer rows.Close()

	var result []PipelineClip
	for rows.Next() {
		c, err := scanPipelineClipRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

// UpdateFilePath updates the file_path for a given clip.
func (r *PipelineClipRepo) UpdateFilePath(ctx context.Context, clipID int64, filePath string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_clips SET file_path = ? WHERE id = ?", filePath, clipID)
	if err != nil {
		return fmt.Errorf("update file_path pipeline_clip %d: %w", clipID, err)
	}
	return nil
}

// UpdateThumbnailPath updates the thumbnail_path and thumbnail_style for a given clip.
func (r *PipelineClipRepo) UpdateThumbnailPath(ctx context.Context, clipID int64, thumbnailPath, thumbnailStyle string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_clips SET thumbnail_path = ?, thumbnail_style = ? WHERE id = ?", thumbnailPath, thumbnailStyle, clipID)
	if err != nil {
		return fmt.Errorf("update thumbnail_path pipeline_clip %d: %w", clipID, err)
	}
	return nil
}

// UpdateMetadata updates the title, description, tags, and thumbnail_text for a given clip.
func (r *PipelineClipRepo) UpdateMetadata(ctx context.Context, clipID int64, title, description, tags, thumbnailText string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_clips SET title = ?, description = ?, tags = ?, thumbnail_text = ? WHERE id = ?",
		title, description, tags, thumbnailText, clipID)
	if err != nil {
		return fmt.Errorf("update metadata pipeline_clip %d: %w", clipID, err)
	}
	return nil
}

// UpdateStatus updates the upload_status for a given clip.
func (r *PipelineClipRepo) UpdateStatus(ctx context.Context, clipID int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_clips SET upload_status = ? WHERE id = ?", status, clipID)
	if err != nil {
		return fmt.Errorf("update upload_status pipeline_clip %d: %w", clipID, err)
	}
	return nil
}

// FindReadyClipsByRunID returns clips with valid file_path and status "ready" for a given run.
func (r *PipelineClipRepo) FindReadyClipsByRunID(ctx context.Context, runID int64) ([]PipelineClip, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+pipelineClipCols+" FROM pipeline_clips WHERE run_id = ? AND file_path != '' AND upload_status = 'ready' ORDER BY start_sec ASC",
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("find ready clips for run %d: %w", runID, err)
	}
	defer rows.Close()

	var result []PipelineClip
	for rows.Next() {
		c, err := scanPipelineClipRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

func scanPipelineClipRows(rows *sql.Rows) (*PipelineClip, error) {
	var c PipelineClip
	err := rows.Scan(&c.ID, &c.RunID, &c.HighlightID, &c.FilePath, &c.ThumbnailPath, &c.Title, &c.Description, &c.Tags, &c.ThumbnailStyle, &c.ThumbnailText, &c.IsSelected, &c.StartSec, &c.EndSec, &c.DurationSec, &c.UploadStatus, &c.YouTubeID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan pipeline_clip: %w", err)
	}
	return &c, nil
}
