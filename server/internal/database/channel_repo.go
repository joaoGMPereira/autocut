package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// ChannelRepo provides CRUD for channels.
// Kotlin ref: ChannelRepository.kt
type ChannelRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewChannelRepo(db *sql.DB, log *slog.Logger) *ChannelRepo {
	return &ChannelRepo{db: db, log: log.With("repo", "channel")}
}

const channelCols = `id, name, channel_id, channel_title, avatar_url, access_token, refresh_token, expires_at, oauth_client_secret_id, is_favorite, created_at, updated_at`

// Create creates a channel with just a name (Kotlin creates with defaults).
// Kotlin ref: create(name: String): Channel
func (r *ChannelRepo) Create(ctx context.Context, name string) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO channels (name, created_at, updated_at) VALUES (?, ?, ?)`,
		name, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert channel: %w", err)
	}
	return res.LastInsertId()
}

func (r *ChannelRepo) GetByID(ctx context.Context, id int64) (*Channel, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+channelCols+" FROM channels WHERE id = ?", id)
	return scanChannel(row)
}

// Kotlin ref: getByName(name: String): Channel?
func (r *ChannelRepo) GetByName(ctx context.Context, name string) (*Channel, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+channelCols+" FROM channels WHERE name = ?", name)
	return scanChannel(row)
}

// Kotlin ref: getByYouTubeChannelId(youtubeChannelId: String): Channel?
func (r *ChannelRepo) GetByYouTubeID(ctx context.Context, ytChannelID string) (*Channel, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+channelCols+" FROM channels WHERE channel_id = ?", ytChannelID)
	return scanChannel(row)
}

// Kotlin ref: getAll(): List<Channel>
func (r *ChannelRepo) List(ctx context.Context) ([]Channel, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+channelCols+" FROM channels ORDER BY is_favorite DESC, name ASC")
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()

	var result []Channel
	for rows.Next() {
		ch, err := scanChannelRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *ch)
	}
	return result, rows.Err()
}

// Kotlin ref: getFavorites(): List<Channel>
func (r *ChannelRepo) ListFavorites(ctx context.Context) ([]Channel, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+channelCols+" FROM channels WHERE is_favorite = 1 ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("list favorite channels: %w", err)
	}
	defer rows.Close()

	var result []Channel
	for rows.Next() {
		ch, err := scanChannelRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *ch)
	}
	return result, rows.Err()
}

// Kotlin ref: update(id, channelId, channelTitle, avatarUrl, accessToken, refreshToken, expiresAt, oauthClientSecretId)
func (r *ChannelRepo) Update(ctx context.Context, ch *Channel) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`UPDATE channels SET
			name = ?, channel_id = ?, channel_title = ?, avatar_url = ?,
			access_token = ?, refresh_token = ?, expires_at = ?,
			oauth_client_secret_id = ?, is_favorite = ?, updated_at = ?
		 WHERE id = ?`,
		ch.Name, ch.ChannelID, ch.ChannelTitle, ch.AvatarURL,
		ch.AccessToken, ch.RefreshToken, ch.ExpiresAt,
		ch.OAuthClientSecretID, ch.IsFavorite, now, ch.ID,
	)
	if err != nil {
		return fmt.Errorf("update channel %d: %w", ch.ID, err)
	}
	return nil
}

// UpdateTokens updates only OAuth tokens (common operation during refresh).
func (r *ChannelRepo) UpdateTokens(ctx context.Context, id int64, accessToken, refreshToken string, expiresAt int64) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`UPDATE channels SET access_token = ?, refresh_token = ?, expires_at = ?, updated_at = ? WHERE id = ?`,
		accessToken, refreshToken, expiresAt, now, id,
	)
	if err != nil {
		return fmt.Errorf("update tokens channel %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: toggleFavorite(id: Int): Boolean
func (r *ChannelRepo) ToggleFavorite(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE channels SET is_favorite = NOT is_favorite, updated_at = ? WHERE id = ?`,
		time.Now().UnixMilli(), id,
	)
	if err != nil {
		return fmt.Errorf("toggle favorite channel %d: %w", id, err)
	}
	return nil
}

func (r *ChannelRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM channels WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete channel %d: %w", id, err)
	}
	return nil
}

// Kotlin ref: delete(name: String): Boolean
func (r *ChannelRepo) DeleteByName(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM channels WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("delete channel %q: %w", name, err)
	}
	return nil
}

func scanChannel(row *sql.Row) (*Channel, error) {
	var ch Channel
	err := row.Scan(
		&ch.ID, &ch.Name, &ch.ChannelID, &ch.ChannelTitle, &ch.AvatarURL,
		&ch.AccessToken, &ch.RefreshToken, &ch.ExpiresAt,
		&ch.OAuthClientSecretID, &ch.IsFavorite, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func scanChannelRows(rows *sql.Rows) (*Channel, error) {
	var ch Channel
	err := rows.Scan(
		&ch.ID, &ch.Name, &ch.ChannelID, &ch.ChannelTitle, &ch.AvatarURL,
		&ch.AccessToken, &ch.RefreshToken, &ch.ExpiresAt,
		&ch.OAuthClientSecretID, &ch.IsFavorite, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}
