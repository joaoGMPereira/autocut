package uploader

import (
	"fmt"
	"log/slog"
	"time"
)

// Quota costs mirror YouTube Data API v3 published values.
// Kotlin ref: QuotaUsage constants in QuotaTrackingDelegate
const (
	QuotaCostUpload        = 1600
	QuotaCostThumbnailSet  = 50
	QuotaCostPlaylistInsert = 50
	QuotaCostVideoUpdate   = 50
	QuotaCostCommentInsert = 50

	quotaDailyLimit = 10_000
)

// QuotaUsageRepository is the storage contract for QuotaTracker.
// The database.QuotaUsageRepo satisfies this interface.
type QuotaUsageRepository interface {
	RecordUsage(channelID, operation string, cost int, date time.Time) error
	UsageByDate(channelID string, date time.Time) (int, error)
}

// QuotaTracker records and queries YouTube API quota usage per channel.
// Kotlin ref: QuotaTrackingDelegate
type QuotaTracker struct {
	repo QuotaUsageRepository
	log  *slog.Logger
}

// NewQuotaTracker creates a QuotaTracker backed by repo.
func NewQuotaTracker(repo QuotaUsageRepository) *QuotaTracker {
	return &QuotaTracker{
		repo: repo,
		log:  slog.With("component", "uploader.quota"),
	}
}

// Track records a quota usage event for channelID.
// Kotlin ref: QuotaTrackingDelegate.recordUpload / recordThumbnail / recordComment
func (q *QuotaTracker) Track(channelID, operation string, cost int) error {
	now := time.Now()
	if err := q.repo.RecordUsage(channelID, operation, cost, now); err != nil {
		return fmt.Errorf("track quota [%s/%s]: %w", channelID, operation, err)
	}
	q.log.Info("quota tracked", "channelID", channelID, "op", operation, "cost", cost)
	return nil
}

// Used returns the total quota consumed by channelID on the given date.
// Kotlin ref: QuotaRepository.getOrCreateTodayUsage().unitsUsed
func (q *QuotaTracker) Used(channelID string, date time.Time) (int, error) {
	used, err := q.repo.UsageByDate(channelID, date)
	if err != nil {
		return 0, fmt.Errorf("get quota used [%s]: %w", channelID, err)
	}
	return used, nil
}

// Remaining returns how many quota units channelID has left today.
// Daily limit is 10 000 units (YouTube default project quota).
// Kotlin ref: QuotaUsage.remainingUnits
func (q *QuotaTracker) Remaining(channelID string) (int, error) {
	used, err := q.Used(channelID, time.Now())
	if err != nil {
		return 0, err
	}
	remaining := quotaDailyLimit - used
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}
