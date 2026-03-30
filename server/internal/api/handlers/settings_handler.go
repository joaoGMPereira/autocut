package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// SettingsHandler handles app-settings CRUD operations.
type SettingsHandler struct {
	repo *database.AppSettingRepo
	log  *slog.Logger
}

// NewSettingsHandler creates a SettingsHandler.
func NewSettingsHandler(db *sql.DB) *SettingsHandler {
	return &SettingsHandler{
		repo: database.NewAppSettingRepo(db, slog.Default()),
		log:  slog.With("component", "api", "handler", "settings"),
	}
}

type updateSettingRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetSettings handles GET /api/settings — returns all settings.
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.List(r.Context())
	if err != nil {
		h.log.Error("list settings failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	if settings == nil {
		settings = []database.AppSetting{}
	}
	writeJSON(w, http.StatusOK, settings)
}

// PutSettings handles PUT /api/settings — upserts a single setting.
func (h *SettingsHandler) PutSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "key is required")
		return
	}

	if err := h.repo.Set(r.Context(), req.Key, req.Value); err != nil {
		h.log.Error("set setting failed", "key", req.Key, "err", err)
		writeError(w, http.StatusInternalServerError, "set_failed", err.Error())
		return
	}

	h.log.Info("setting updated", "key", req.Key)
	writeJSON(w, http.StatusOK, map[string]string{"key": req.Key, "value": req.Value})
}
