package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// URLHistoryHandler handles HTTP requests for URL history management.
type URLHistoryHandler struct {
	repo *database.URLHistoryRepo
	log  *slog.Logger
}

// NewURLHistoryHandler constructs a URLHistoryHandler.
func NewURLHistoryHandler(repo *database.URLHistoryRepo) *URLHistoryHandler {
	return &URLHistoryHandler{
		repo: repo,
		log:  slog.With("handler", "url_history"),
	}
}

// GetHistory returns the URL history list ordered by most recently used.
// GET /api/url-history
func (h *URLHistoryHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	entries, err := h.repo.List(r.Context())
	if err != nil {
		h.log.Error("list url_history failed", "err", err)
		jsonError(w, "failed to list url history", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []database.URLHistoryEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"history": entries})
}

// DeleteEntry removes a single URL history entry by ID.
// DELETE /api/url-history/{id}
func (h *URLHistoryHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		jsonError(w, "invalid history entry id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, "entry not found", http.StatusNotFound)
			return
		}
		h.log.Error("delete url_history entry failed", "id", id, "err", err)
		jsonError(w, "failed to delete entry", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteAll clears all URL history entries.
// DELETE /api/url-history
func (h *URLHistoryHandler) DeleteAll(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.DeleteAll(r.Context()); err != nil {
		h.log.Error("delete all url_history failed", "err", err)
		jsonError(w, "failed to clear url history", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
