package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ChannelConfigHandler handles channel configuration endpoints.
type ChannelConfigHandler struct {
	repo *database.ChannelConfigRepo
	log  *slog.Logger
}

// NewChannelConfigHandler creates a ChannelConfigHandler.
func NewChannelConfigHandler(repo *database.ChannelConfigRepo) *ChannelConfigHandler {
	return &ChannelConfigHandler{
		repo: repo,
		log:  slog.With("component", "api", "handler", "channel_config"),
	}
}

// GetConfig handles GET /api/channels/{id}/config.
func (h *ChannelConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	if rawID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}

	channelID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "id must be an integer")
		return
	}

	cfg, err := h.repo.GetByChannelID(r.Context(), channelID)
	if err != nil {
		h.log.Error("get channel config failed", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig handles PUT /api/channels/{id}/config.
func (h *ChannelConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	if rawID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}

	channelID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "id must be an integer")
		return
	}

	var cfg database.ChannelConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	cfg.ChannelID = channelID

	if err := h.repo.Update(r.Context(), &cfg); err != nil {
		h.log.Error("update channel config failed", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}
