package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// AppSettingRepo provides CRUD for app_settings (key-value store).
// Kotlin ref: AppSettingsRepository.kt
type AppSettingRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewAppSettingRepo(db *sql.DB, log *slog.Logger) *AppSettingRepo {
	return &AppSettingRepo{db: db, log: log.With("repo", "app_setting")}
}

// Get returns the value for the given key, or empty string if not found.
// Kotlin ref: get(key: String): String?
func (r *AppSettingRepo) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		"SELECT value FROM app_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

// Set creates or updates a setting. Uses UPSERT (ON CONFLICT).
// Kotlin ref: set(key: String, value: String): Boolean
func (r *AppSettingRepo) Set(ctx context.Context, key, value string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, now,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

// List returns all settings.
// Kotlin ref: getAll(): Map<String, String>
func (r *AppSettingRepo) List(ctx context.Context) ([]AppSetting, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, key, value, updated_at FROM app_settings ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("list app_settings: %w", err)
	}
	defer rows.Close()

	var result []AppSetting
	for rows.Next() {
		var s AppSetting
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// Delete removes a setting by key.
// Kotlin ref: delete(key: String): Boolean
func (r *AppSettingRepo) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM app_settings WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("delete setting %q: %w", key, err)
	}
	return nil
}
