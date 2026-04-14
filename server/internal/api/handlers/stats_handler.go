package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/stats"
)

// StatsProvider is the interface the StatsHandler depends on.
type StatsProvider interface {
	Get() stats.Data
}

// StatsHandler serves the /api/stats endpoint.
type StatsHandler struct {
	svc StatsProvider
}

// NewStatsHandler creates a StatsHandler.
func NewStatsHandler(svc StatsProvider) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// GetStats returns aggregate stats for the dashboard.
func (h *StatsHandler) GetStats(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.svc.Get())
}
