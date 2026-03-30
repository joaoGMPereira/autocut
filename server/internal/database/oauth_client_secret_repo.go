package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// OAuthClientSecretRepo provides CRUD for oauth_client_secrets.
// Kotlin ref: OAuthClientSecretRepository.kt
type OAuthClientSecretRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewOAuthClientSecretRepo(db *sql.DB, log *slog.Logger) *OAuthClientSecretRepo {
	return &OAuthClientSecretRepo{db: db, log: log.With("repo", "oauth_secret")}
}

func (r *OAuthClientSecretRepo) Create(ctx context.Context, o *OAuthClientSecret) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO oauth_client_secrets (name, client_id, client_secret, project_id, is_default, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		o.Name, o.ClientID, o.ClientSecret, o.ProjectID, o.IsDefault, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert oauth_client_secret: %w", err)
	}
	return res.LastInsertId()
}

func (r *OAuthClientSecretRepo) GetByID(ctx context.Context, id int64) (*OAuthClientSecret, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, client_id, client_secret, project_id, is_default, created_at, updated_at
		 FROM oauth_client_secrets WHERE id = ?`, id)
	return scanOAuthClientSecret(row)
}

func (r *OAuthClientSecretRepo) GetByName(ctx context.Context, name string) (*OAuthClientSecret, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, client_id, client_secret, project_id, is_default, created_at, updated_at
		 FROM oauth_client_secrets WHERE name = ?`, name)
	return scanOAuthClientSecret(row)
}

func (r *OAuthClientSecretRepo) GetDefault(ctx context.Context) (*OAuthClientSecret, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, client_id, client_secret, project_id, is_default, created_at, updated_at
		 FROM oauth_client_secrets WHERE is_default = 1 LIMIT 1`)
	return scanOAuthClientSecret(row)
}

func (r *OAuthClientSecretRepo) List(ctx context.Context) ([]OAuthClientSecret, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, client_id, client_secret, project_id, is_default, created_at, updated_at
		 FROM oauth_client_secrets ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list oauth_client_secrets: %w", err)
	}
	defer rows.Close()

	var result []OAuthClientSecret
	for rows.Next() {
		o, err := scanOAuthClientSecretRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *o)
	}
	return result, rows.Err()
}

func (r *OAuthClientSecretRepo) Update(ctx context.Context, o *OAuthClientSecret) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`UPDATE oauth_client_secrets
		 SET name = ?, client_id = ?, client_secret = ?, project_id = ?, is_default = ?, updated_at = ?
		 WHERE id = ?`,
		o.Name, o.ClientID, o.ClientSecret, o.ProjectID, o.IsDefault, now, o.ID,
	)
	if err != nil {
		return fmt.Errorf("update oauth_client_secret %d: %w", o.ID, err)
	}
	return nil
}

// SetDefault clears all defaults, then sets the given id as default.
// Kotlin ref: setAsDefault(id) — uses transaction
func (r *OAuthClientSecretRepo) SetDefault(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "UPDATE oauth_client_secrets SET is_default = 0"); err != nil {
		tx.Rollback()
		return fmt.Errorf("clear defaults: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "UPDATE oauth_client_secrets SET is_default = 1 WHERE id = ?", id); err != nil {
		tx.Rollback()
		return fmt.Errorf("set default %d: %w", id, err)
	}

	return tx.Commit()
}

func (r *OAuthClientSecretRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM oauth_client_secrets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete oauth_client_secret %d: %w", id, err)
	}
	return nil
}

func scanOAuthClientSecret(row *sql.Row) (*OAuthClientSecret, error) {
	var o OAuthClientSecret
	err := row.Scan(&o.ID, &o.Name, &o.ClientID, &o.ClientSecret, &o.ProjectID, &o.IsDefault, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func scanOAuthClientSecretRows(rows *sql.Rows) (*OAuthClientSecret, error) {
	var o OAuthClientSecret
	err := rows.Scan(&o.ID, &o.Name, &o.ClientID, &o.ClientSecret, &o.ProjectID, &o.IsDefault, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
