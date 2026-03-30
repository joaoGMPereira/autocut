package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ChannelHandler handles channel CRUD operations.
type ChannelHandler struct {
	repo *database.ChannelRepo
	log  *slog.Logger
}

// NewChannelHandler creates a ChannelHandler.
func NewChannelHandler(db *sql.DB) *ChannelHandler {
	return &ChannelHandler{
		repo: database.NewChannelRepo(db, slog.Default()),
		log:  slog.With("component", "api", "handler", "channel"),
	}
}

type createChannelRequest struct {
	Name            string `json:"name"`
	YouTubeChannelID string `json:"youtube_channel_id"`
}

// GetChannels handles GET /api/channels.
func (h *ChannelHandler) GetChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.repo.List(r.Context())
	if err != nil {
		h.log.Error("list channels failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	if channels == nil {
		channels = []database.Channel{}
	}
	writeJSON(w, http.StatusOK, channels)
}

// PostChannel handles POST /api/channels.
func (h *ChannelHandler) PostChannel(w http.ResponseWriter, r *http.Request) {
	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "name is required")
		return
	}

	id, err := h.repo.Create(r.Context(), req.Name)
	if err != nil {
		h.log.Error("create channel failed", "err", err)
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	// If a YouTube channel ID was provided, update the record immediately.
	if req.YouTubeChannelID != "" {
		ch, getErr := h.repo.GetByID(r.Context(), id)
		if getErr == nil {
			ch.ChannelID = req.YouTubeChannelID
			_ = h.repo.Update(r.Context(), ch)
		}
	}

	ch, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}

	h.log.Info("channel created", "id", id, "name", req.Name)
	writeJSON(w, http.StatusCreated, ch)
}

// DeleteChannel handles DELETE /api/channels/{id}.
func (h *ChannelHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	if rawID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "id must be an integer")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.log.Error("delete channel failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	h.log.Info("channel deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
