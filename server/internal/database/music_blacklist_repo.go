package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// MusicBlacklistRepo provides CRUD for the music_blacklist table.
// FP-009: per-channel music pattern exclusion list.
type MusicBlacklistRepo struct {
	db  *sql.DB
	log *slog.Logger
}

// NewMusicBlacklistRepo creates a MusicBlacklistRepo.
func NewMusicBlacklistRepo(db *sql.DB, log *slog.Logger) *MusicBlacklistRepo {
	return &MusicBlacklistRepo{db: db, log: log.With("repo", "music_blacklist")}
}

// ListByChannel returns all blacklist entries for a channel, ordered by id.
func (r *MusicBlacklistRepo) ListByChannel(ctx context.Context, channelID int64) ([]MusicBlacklistEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, channel_id, pattern, created_at FROM music_blacklist WHERE channel_id = ? ORDER BY id",
		channelID,
	)
	if err != nil {
		return nil, fmt.Errorf("list music_blacklist for channel %d: %w", channelID, err)
	}
	defer rows.Close()

	var entries []MusicBlacklistEntry
	for rows.Next() {
		var e MusicBlacklistEntry
		if err := rows.Scan(&e.ID, &e.ChannelID, &e.Pattern, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan music_blacklist row: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate music_blacklist rows: %w", err)
	}
	return entries, nil
}

// Create inserts a new blacklist entry and returns the created record.
func (r *MusicBlacklistRepo) Create(ctx context.Context, channelID int64, pattern string) (*MusicBlacklistEntry, error) {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO music_blacklist (channel_id, pattern) VALUES (?, ?)",
		channelID, pattern,
	)
	if err != nil {
		return nil, fmt.Errorf("insert music_blacklist for channel %d: %w", channelID, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id for music_blacklist: %w", err)
	}

	var entry MusicBlacklistEntry
	if err := r.db.QueryRowContext(ctx,
		"SELECT id, channel_id, pattern, created_at FROM music_blacklist WHERE id = ?", id,
	).Scan(&entry.ID, &entry.ChannelID, &entry.Pattern, &entry.CreatedAt); err != nil {
		return nil, fmt.Errorf("fetch created music_blacklist entry %d: %w", id, err)
	}
	return &entry, nil
}

// Delete removes a blacklist entry by its id.
func (r *MusicBlacklistRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM music_blacklist WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete music_blacklist entry %d: %w", id, err)
	}
	return nil
}
