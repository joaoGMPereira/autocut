package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// QuotaUsageRepo provides CRUD for quota_usage (YouTube API quota tracking).
// Kotlin ref: QuotaRepository.kt
type QuotaUsageRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewQuotaUsageRepo(db *sql.DB, log *slog.Logger) *QuotaUsageRepo {
	return &QuotaUsageRepo{db: db, log: log.With("repo", "quota_usage")}
}

const quotaCols = `id, oauth_client_secret_id, date, units_used, upload_count, thumbnail_count, other_api_calls, created_at, updated_at`

// GetOrCreateToday returns today's usage for a secret, creating it if needed.
// Kotlin ref: getOrCreateTodayUsage(clientSecretId)
func (r *QuotaUsageRepo) GetOrCreateToday(ctx context.Context, secretID int64) (*QuotaUsage, error) {
	date := currentDatePT()
	now := time.Now().UnixMilli()

	// UPSERT: insert if not exists, return existing
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO quota_usage (oauth_client_secret_id, date, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(oauth_client_secret_id, date) DO NOTHING`,
		secretID, date, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert quota_usage: %w", err)
	}

	return r.GetBySecretAndDate(ctx, secretID, date)
}

func (r *QuotaUsageRepo) GetByID(ctx context.Context, id int64) (*QuotaUsage, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+quotaCols+" FROM quota_usage WHERE id = ?", id)
	return scanQuotaUsage(row)
}

// Kotlin ref: getBySecretAndDate — used internally
func (r *QuotaUsageRepo) GetBySecretAndDate(ctx context.Context, secretID int64, date string) (*QuotaUsage, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+quotaCols+" FROM quota_usage WHERE oauth_client_secret_id = ? AND date = ?",
		secretID, date)
	return scanQuotaUsage(row)
}

// Kotlin ref: getAllTodayUsage(): List<QuotaUsage>
func (r *QuotaUsageRepo) ListToday(ctx context.Context) ([]QuotaUsage, error) {
	date := currentDatePT()
	return r.queryQuota(ctx,
		"SELECT "+quotaCols+" FROM quota_usage WHERE date = ?", date)
}

// Kotlin ref: getUsageHistory(clientSecretId, days): List<QuotaUsage>
func (r *QuotaUsageRepo) ListHistory(ctx context.Context, secretID int64, days int) ([]QuotaUsage, error) {
	return r.queryQuota(ctx,
		"SELECT "+quotaCols+" FROM quota_usage WHERE oauth_client_secret_id = ? ORDER BY date DESC LIMIT ?",
		secretID, days)
}

func (r *QuotaUsageRepo) ListBySecret(ctx context.Context, secretID int64) ([]QuotaUsage, error) {
	return r.queryQuota(ctx,
		"SELECT "+quotaCols+" FROM quota_usage WHERE oauth_client_secret_id = ? ORDER BY date DESC",
		secretID)
}

// RecordUpload adds 1600 units + increments upload count.
// Kotlin ref: recordUpload(clientSecretId)
func (r *QuotaUsageRepo) RecordUpload(ctx context.Context, secretID int64) error {
	date := currentDatePT()
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO quota_usage (oauth_client_secret_id, date, units_used, upload_count, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)
		 ON CONFLICT(oauth_client_secret_id, date) DO UPDATE SET
			units_used = units_used + ?, upload_count = upload_count + 1, updated_at = ?`,
		secretID, date, QuotaUploadCost, now, now, QuotaUploadCost, now,
	)
	if err != nil {
		return fmt.Errorf("record upload quota: %w", err)
	}
	return nil
}

// RecordThumbnail adds 50 units + increments thumbnail count.
// Kotlin ref: recordThumbnail(clientSecretId)
func (r *QuotaUsageRepo) RecordThumbnail(ctx context.Context, secretID int64) error {
	date := currentDatePT()
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO quota_usage (oauth_client_secret_id, date, units_used, thumbnail_count, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)
		 ON CONFLICT(oauth_client_secret_id, date) DO UPDATE SET
			units_used = units_used + ?, thumbnail_count = thumbnail_count + 1, updated_at = ?`,
		secretID, date, QuotaThumbnailCost, now, now, QuotaThumbnailCost, now,
	)
	if err != nil {
		return fmt.Errorf("record thumbnail quota: %w", err)
	}
	return nil
}

// RecordAPICall adds arbitrary cost units.
// Kotlin ref: recordApiCall(clientSecretId, cost)
func (r *QuotaUsageRepo) RecordAPICall(ctx context.Context, secretID int64, cost int) error {
	date := currentDatePT()
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO quota_usage (oauth_client_secret_id, date, units_used, other_api_calls, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)
		 ON CONFLICT(oauth_client_secret_id, date) DO UPDATE SET
			units_used = units_used + ?, other_api_calls = other_api_calls + 1, updated_at = ?`,
		secretID, date, cost, now, now, cost, now,
	)
	if err != nil {
		return fmt.Errorf("record api call quota: %w", err)
	}
	return nil
}

// GetRemaining returns remaining quota units for today.
// Kotlin ref: getRemainingQuota(clientSecretId)
func (r *QuotaUsageRepo) GetRemaining(ctx context.Context, secretID int64) (int, error) {
	q, err := r.GetOrCreateToday(ctx, secretID)
	if err != nil {
		return 0, err
	}
	remaining := QuotaDailyLimit - q.UnitsUsed
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// CanUpload checks if there's enough quota for `count` uploads.
// Kotlin ref: canUpload(clientSecretId, count)
func (r *QuotaUsageRepo) CanUpload(ctx context.Context, secretID int64, count int) (bool, error) {
	remaining, err := r.GetRemaining(ctx, secretID)
	if err != nil {
		return false, err
	}
	return remaining >= count*QuotaUploadCost, nil
}

// Cleanup removes records older than daysToKeep.
// Kotlin ref: cleanupOldRecords(daysToKeep)
func (r *QuotaUsageRepo) Cleanup(ctx context.Context, daysToKeep int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -daysToKeep).Format("2006-01-02")
	res, err := r.db.ExecContext(ctx,
		"DELETE FROM quota_usage WHERE date < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup quota_usage: %w", err)
	}
	return res.RowsAffected()
}

func (r *QuotaUsageRepo) queryQuota(ctx context.Context, query string, args ...any) ([]QuotaUsage, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query quota_usage: %w", err)
	}
	defer rows.Close()

	var result []QuotaUsage
	for rows.Next() {
		var q QuotaUsage
		if err := rows.Scan(&q.ID, &q.OAuthClientSecretID, &q.Date, &q.UnitsUsed, &q.UploadCount, &q.ThumbnailCount, &q.OtherAPICalls, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, q)
	}
	return result, rows.Err()
}

func scanQuotaUsage(row *sql.Row) (*QuotaUsage, error) {
	var q QuotaUsage
	err := row.Scan(&q.ID, &q.OAuthClientSecretID, &q.Date, &q.UnitsUsed, &q.UploadCount, &q.ThumbnailCount, &q.OtherAPICalls, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// currentDatePT returns the current date in Pacific Time (YYYY-MM-DD).
// Kotlin ref: getCurrentDatePT() — YouTube quota resets at midnight PT.
func currentDatePT() string {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		// Fallback to UTC if timezone not available
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().In(loc).Format("2006-01-02")
}
