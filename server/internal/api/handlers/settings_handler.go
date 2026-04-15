package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// AppSettingRepoFace is the interface SettingsHandler depends on.
type AppSettingRepoFace interface {
	List(ctx context.Context) ([]database.AppSetting, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

// SettingsHandler handles app-settings CRUD.
type SettingsHandler struct {
	repo AppSettingRepoFace
}

// NewSettingsHandler creates a SettingsHandler backed by repo.
func NewSettingsHandler(repo AppSettingRepoFace) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

// GetSettings returns all app settings as a JSON array.
// GET /api/settings → 200 []AppSetting
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.repo.List(r.Context())
	if err != nil {
		slog.Error("list settings failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []database.AppSetting{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rows)
}

// PutSetting creates or updates a single setting.
// PUT /api/settings body: {key, value} → 200 {key, value}
func (h *SettingsHandler) PutSetting(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "invalid_request", "message": "invalid JSON body"})
		return
	}
	if body.Key == "" || body.Value == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "invalid_request", "message": "key and value are required"})
		return
	}

	if err := h.repo.Set(r.Context(), body.Key, body.Value); err != nil {
		slog.Error("set setting failed", "key", body.Key, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("setting updated", "key", body.Key)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"key": body.Key, "value": body.Value})
}

// DeleteSetting removes a single setting by key.
// DELETE /api/settings?key=xxx → 204
func (h *SettingsHandler) DeleteSetting(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "invalid_request", "message": "key query param is required"})
		return
	}
	if err := h.repo.Delete(r.Context(), key); err != nil {
		slog.Error("delete setting failed", "key", key, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	slog.Info("setting deleted", "key", key)
	w.WriteHeader(http.StatusNoContent)
}
