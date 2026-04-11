package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"
)

// QuotaHandler handles quota history and reset endpoints.
type QuotaHandler struct {
	db  *sql.DB
	log *slog.Logger
}

// NewQuotaHandler creates a QuotaHandler.
func NewQuotaHandler(db *sql.DB) *QuotaHandler {
	return &QuotaHandler{
		db:  db,
		log: slog.With("component", "api", "handler", "quota"),
	}
}

// QuotaHistoryEntry represents one day of aggregated quota usage.
type QuotaHistoryEntry struct {
	Date        string `json:"date"`
	UnitsUsed   int    `json:"units_used"`
	UploadCount int    `json:"upload_count"`
}

// quotaTodayDatePT returns today's date in Pacific Time (YYYY-MM-DD).
// Duplicated from stats_handler to keep handlers self-contained.
func quotaTodayDatePT() string {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// GetQuotaHistory handles GET /api/quota/history
// Returns the last 30 days of quota usage, grouped by date and summed across all secrets.
func (h *QuotaHandler) GetQuotaHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.db.QueryContext(ctx,
		`SELECT date, COALESCE(SUM(units_used), 0), COALESCE(SUM(upload_count), 0)
		 FROM quota_usage
		 GROUP BY date
		 ORDER BY date DESC
		 LIMIT 30`,
	)
	if err != nil {
		h.log.Error("quota history: query failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}
	defer rows.Close()

	history := make([]QuotaHistoryEntry, 0)
	for rows.Next() {
		var entry QuotaHistoryEntry
		if err := rows.Scan(&entry.Date, &entry.UnitsUsed, &entry.UploadCount); err != nil {
			h.log.Error("quota history: scan failed", "err", err)
			writeError(w, http.StatusInternalServerError, "scan_failed", err.Error())
			return
		}
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		h.log.Error("quota history: iterate failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	h.log.Info("quota history fetched", "days", len(history))
	writeJSON(w, http.StatusOK, map[string]interface{}{"history": history})
}

// PostQuotaReset handles POST /api/quota/reset
// Resets today's quota_usage counters to zero across all secrets.
func (h *QuotaHandler) PostQuotaReset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UnixMilli()
	today := quotaTodayDatePT()

	_, err := h.db.ExecContext(ctx,
		`UPDATE quota_usage
		 SET units_used = 0, upload_count = 0, thumbnail_count = 0, other_api_calls = 0, updated_at = ?
		 WHERE date = ?`,
		now, today,
	)
	if err != nil {
		h.log.Error("quota reset: update failed", "err", err)
		writeError(w, http.StatusInternalServerError, "reset_failed", err.Error())
		return
	}

	h.log.Info("quota reset for today", "date", today)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
