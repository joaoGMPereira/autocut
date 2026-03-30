package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// ThumbnailBackgroundRepo provides CRUD for thumbnail_backgrounds.
// Kotlin ref: ThumbnailBackgroundRepository.kt
type ThumbnailBackgroundRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewThumbnailBackgroundRepo(db *sql.DB, log *slog.Logger) *ThumbnailBackgroundRepo {
	return &ThumbnailBackgroundRepo{db: db, log: log.With("repo", "thumb_bg")}
}

const thumbBgCols = `id, channel_id, name, file_path, is_default, created_at`

func (r *ThumbnailBackgroundRepo) Create(ctx context.Context, tb *ThumbnailBackground) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO thumbnail_backgrounds (channel_id, name, file_path, is_default, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		tb.ChannelID, tb.Name, tb.FilePath, tb.IsDefault, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert thumbnail_background: %w", err)
	}
	return res.LastInsertId()
}

func (r *ThumbnailBackgroundRepo) GetByID(ctx context.Context, id int64) (*ThumbnailBackground, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+thumbBgCols+" FROM thumbnail_backgrounds WHERE id = ?", id)
	return scanThumbBg(row)
}

// Kotlin ref: getDefaultForChannel(channelId): ThumbnailBackground?
func (r *ThumbnailBackgroundRepo) GetDefault(ctx context.Context, channelID int64) (*ThumbnailBackground, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+thumbBgCols+" FROM thumbnail_backgrounds WHERE channel_id = ? AND is_default = 1 LIMIT 1",
		channelID)
	return scanThumbBg(row)
}

// Kotlin ref: getByChannelId(channelId): List<ThumbnailBackground>
func (r *ThumbnailBackgroundRepo) ListByChannel(ctx context.Context, channelID int64) ([]ThumbnailBackground, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+thumbBgCols+" FROM thumbnail_backgrounds WHERE channel_id = ? ORDER BY created_at DESC",
		channelID)
	if err != nil {
		return nil, fmt.Errorf("list thumb_bg by channel: %w", err)
	}
	defer rows.Close()

	var result []ThumbnailBackground
	for rows.Next() {
		tb, err := scanThumbBgRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *tb)
	}
	return result, rows.Err()
}

// SetDefault clears all defaults for a channel, then sets the given bg as default.
// Kotlin ref: setDefault(id, channelId): Boolean
func (r *ThumbnailBackgroundRepo) SetDefault(ctx context.Context, id, channelID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE thumbnail_backgrounds SET is_default = 0 WHERE channel_id = ?", channelID); err != nil {
		tx.Rollback()
		return fmt.Errorf("clear defaults: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE thumbnail_backgrounds SET is_default = 1 WHERE id = ? AND channel_id = ?", id, channelID); err != nil {
		tx.Rollback()
		return fmt.Errorf("set default %d: %w", id, err)
	}

	return tx.Commit()
}

func (r *ThumbnailBackgroundRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM thumbnail_backgrounds WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete thumb_bg %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: deleteAllForChannel(channelId): Int
func (r *ThumbnailBackgroundRepo) DeleteByChannel(ctx context.Context, channelID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM thumbnail_backgrounds WHERE channel_id = ?", channelID)
	if err != nil {
		return fmt.Errorf("delete thumb_bg by channel %d: %w", channelID, err)
	}
	return nil
}

func scanThumbBg(row *sql.Row) (*ThumbnailBackground, error) {
	var tb ThumbnailBackground
	err := row.Scan(&tb.ID, &tb.ChannelID, &tb.Name, &tb.FilePath, &tb.IsDefault, &tb.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &tb, nil
}

func scanThumbBgRows(rows *sql.Rows) (*ThumbnailBackground, error) {
	var tb ThumbnailBackground
	err := rows.Scan(&tb.ID, &tb.ChannelID, &tb.Name, &tb.FilePath, &tb.IsDefault, &tb.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &tb, nil
}
