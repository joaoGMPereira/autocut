package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// URLHistoryRepo provides CRUD for the url_history table (migration v11).
type URLHistoryRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewURLHistoryRepo(db *sql.DB, log *slog.Logger) *URLHistoryRepo {
	return &URLHistoryRepo{db: db, log: log.With("repo", "url_history")}
}

// List returns up to 10 URL history entries ordered by most recently used.
func (r *URLHistoryRepo) List(ctx context.Context) ([]URLHistoryEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, url, video_title, last_used_at, use_count
		 FROM url_history ORDER BY last_used_at DESC LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("list url_history: %w", err)
	}
	defer rows.Close()

	var result []URLHistoryEntry
	for rows.Next() {
		var e URLHistoryEntry
		if err := rows.Scan(&e.ID, &e.URL, &e.VideoTitle, &e.LastUsedAt, &e.UseCount); err != nil {
			return nil, fmt.Errorf("scan url_history row: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// Upsert inserts a new URL history entry or updates it if the URL already exists.
// After upsert, evicts entries beyond the top-10 most recently used.
func (r *URLHistoryRepo) Upsert(ctx context.Context, e URLHistoryEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO url_history (url, video_title, last_used_at, use_count)
		 VALUES (?, ?, ?, 1)
		 ON CONFLICT(url) DO UPDATE SET
		   video_title  = excluded.video_title,
		   last_used_at = excluded.last_used_at,
		   use_count    = url_history.use_count + 1`,
		e.URL, e.VideoTitle, e.LastUsedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert url_history %q: %w", e.URL, err)
	}
	// Evict entries beyond top-10.
	_, err = r.db.ExecContext(ctx,
		`DELETE FROM url_history
		 WHERE id NOT IN (
		   SELECT id FROM url_history ORDER BY last_used_at DESC LIMIT 10
		 )`)
	if err != nil {
		return fmt.Errorf("evict url_history: %w", err)
	}
	return nil
}

// Delete removes a single entry by ID. Returns nil if not found (404-safe).
func (r *URLHistoryRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM url_history WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete url_history %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteAll removes all URL history entries.
func (r *URLHistoryRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM url_history")
	if err != nil {
		return fmt.Errorf("delete all url_history: %w", err)
	}
	return nil
}
