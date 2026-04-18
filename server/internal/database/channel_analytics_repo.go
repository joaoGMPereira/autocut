package database

import (
	"context"
	"database/sql"
	"time"
)

// ChannelAnalyticsRepo manages the channel_analytics cache table.
type ChannelAnalyticsRepo struct {
	db *sql.DB
}

// NewChannelAnalyticsRepo constructs a ChannelAnalyticsRepo.
func NewChannelAnalyticsRepo(db *sql.DB) *ChannelAnalyticsRepo {
	return &ChannelAnalyticsRepo{db: db}
}

// Get returns cached analytics for channelID if the row exists and fetched_at
// is within ttlSec seconds of now. Returns nil (no error) when the cache is
// missing or stale.
func (r *ChannelAnalyticsRepo) Get(ctx context.Context, channelID int64, ttlSec int64) (*ChannelAnalytics, error) {
	threshold := time.Now().Unix() - ttlSec
	row := r.db.QueryRowContext(ctx,
		`SELECT id, channel_id, fetched_at, video_count, avg_views,
                top_title_words, top_tags, success_titles, raw_titles
         FROM channel_analytics
         WHERE channel_id = ? AND fetched_at > ?`,
		channelID, threshold,
	)
	a := &ChannelAnalytics{}
	err := row.Scan(&a.ID, &a.ChannelID, &a.FetchedAt, &a.VideoCount, &a.AvgViews,
		&a.TopTitleWords, &a.TopTags, &a.SuccessTitles, &a.RawTitles)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// Upsert inserts or replaces the analytics row for channelID.
func (r *ChannelAnalyticsRepo) Upsert(ctx context.Context, a *ChannelAnalytics) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO channel_analytics
             (channel_id, fetched_at, video_count, avg_views,
              top_title_words, top_tags, success_titles, raw_titles)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT(channel_id) DO UPDATE SET
             fetched_at      = excluded.fetched_at,
             video_count     = excluded.video_count,
             avg_views       = excluded.avg_views,
             top_title_words = excluded.top_title_words,
             top_tags        = excluded.top_tags,
             success_titles  = excluded.success_titles,
             raw_titles      = excluded.raw_titles`,
		a.ChannelID, a.FetchedAt, a.VideoCount, a.AvgViews,
		a.TopTitleWords, a.TopTags, a.SuccessTitles, a.RawTitles,
	)
	return err
}
