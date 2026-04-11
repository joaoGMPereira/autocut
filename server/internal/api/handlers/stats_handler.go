package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"
)

// StatsHandler handles GET /api/stats — dashboard statistics.
type StatsHandler struct {
	db  *sql.DB
	log *slog.Logger
}

// NewStatsHandler creates a StatsHandler.
func NewStatsHandler(db *sql.DB) *StatsHandler {
	return &StatsHandler{
		db:  db,
		log: slog.With("component", "api", "handler", "stats"),
	}
}

// QuotaBreakdown provides per-operation quota detail for today.
type QuotaBreakdown struct {
	UploadsUsed  int `json:"uploads_used"`
	UploadsLimit int `json:"uploads_limit"`
	UpdatesUsed  int `json:"updates_used"`
	UpdatesLimit int `json:"updates_limit"`
	ReadsUsed    int `json:"reads_used"`
	ReadsLimit   int `json:"reads_limit"`
	TotalUnits   int `json:"total_units"`
	TotalLimit   int `json:"total_limit"`
}

// StatsResponse is the JSON response for GET /api/stats.
type StatsResponse struct {
	ActiveDownloads    int                  `json:"active_downloads"`
	PendingUploads     int                  `json:"pending_uploads"`
	TotalClips         int                  `json:"total_clips"`
	TotalShorts        int                  `json:"total_shorts"`
	ChannelsCount      int                  `json:"channels_count"`
	QuotaUsedToday     int                  `json:"quota_used_today"`
	RecentPipelineRuns []PipelineRunSummary `json:"recent_pipeline_runs"`
	QuotaBreakdown     QuotaBreakdown       `json:"quota_breakdown"`
}

// PipelineRunSummary is a compact view of a pipeline run for the dashboard.
type PipelineRunSummary struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Mode       string `json:"mode"`
	StartedAt  int64  `json:"started_at"`
	FinishedAt *int64 `json:"finished_at"`
}

// statsTodayDate returns today's date in Pacific Time (YYYY-MM-DD).
// YouTube quota resets at midnight PT; inlined here to avoid cross-package coupling.
func statsTodayDate() string {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// GetStats handles GET /api/stats.
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Pending uploads (status = 'queued')
	var pendingUploads int
	if err := h.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM uploads WHERE status = 'queued'",
	).Scan(&pendingUploads); err != nil {
		h.log.Error("stats: count pending uploads failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// 2. Total clips
	var totalClips int
	if err := h.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM clips",
	).Scan(&totalClips); err != nil {
		h.log.Error("stats: count clips failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// 3. Total shorts
	var totalShorts int
	if err := h.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM shorts",
	).Scan(&totalShorts); err != nil {
		h.log.Error("stats: count shorts failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// 4. Channels count
	var channelsCount int
	if err := h.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM channels",
	).Scan(&channelsCount); err != nil {
		h.log.Error("stats: count channels failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// 5. Quota used today (sum across all secrets, COALESCE to 0 if empty)
	var quotaUsedToday int
	if err := h.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(units_used), 0) FROM quota_usage WHERE date = ?",
		statsTodayDate(),
	).Scan(&quotaUsedToday); err != nil {
		h.log.Error("stats: sum quota today failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// 6. Recent pipeline runs (last 5)
	rows, err := h.db.QueryContext(ctx,
		"SELECT id, state, mode, started_at, finished_at FROM pipeline_runs ORDER BY id DESC LIMIT 5",
	)
	if err != nil {
		h.log.Error("stats: query pipeline runs failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}
	defer rows.Close()

	recentRuns := make([]PipelineRunSummary, 0)
	for rows.Next() {
		var run PipelineRunSummary
		var finishedAt sql.NullInt64
		if err := rows.Scan(&run.ID, &run.Status, &run.Mode, &run.StartedAt, &finishedAt); err != nil {
			h.log.Error("stats: scan pipeline run failed", "err", err)
			writeError(w, http.StatusInternalServerError, "scan_failed", err.Error())
			return
		}
		if finishedAt.Valid {
			v := finishedAt.Int64
			run.FinishedAt = &v
		}
		recentRuns = append(recentRuns, run)
	}
	if err := rows.Err(); err != nil {
		h.log.Error("stats: iterate pipeline runs failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// 7. Quota breakdown for today (sum across all oauth_client_secrets)
	var uploadsUsed, updatesUsed, readsUsed int
	if err := h.db.QueryRowContext(ctx,
		`SELECT
			COALESCE(SUM(upload_count), 0),
			COALESCE(SUM(thumbnail_count), 0),
			COALESCE(SUM(other_api_calls), 0)
		FROM quota_usage WHERE date = ?`,
		statsTodayDate(),
	).Scan(&uploadsUsed, &updatesUsed, &readsUsed); err != nil {
		h.log.Error("stats: quota breakdown query failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	breakdown := QuotaBreakdown{
		UploadsUsed:  uploadsUsed,
		UploadsLimit: 1600,
		UpdatesUsed:  updatesUsed,
		UpdatesLimit: 50,
		ReadsUsed:    readsUsed,
		ReadsLimit:   10000,
		TotalUnits:   quotaUsedToday,
		TotalLimit:   10000,
	}

	resp := StatsResponse{
		ActiveDownloads:    0, // SSE hub active job count — static until hub exposes count
		PendingUploads:     pendingUploads,
		TotalClips:         totalClips,
		TotalShorts:        totalShorts,
		ChannelsCount:      channelsCount,
		QuotaUsedToday:     quotaUsedToday,
		RecentPipelineRuns: recentRuns,
		QuotaBreakdown:     breakdown,
	}

	h.log.Info("stats fetched",
		"pending_uploads", pendingUploads,
		"total_clips", totalClips,
		"total_shorts", totalShorts,
		"channels", channelsCount,
		"quota_today", quotaUsedToday,
		"recent_runs", len(recentRuns),
	)

	writeJSON(w, http.StatusOK, resp)
}
