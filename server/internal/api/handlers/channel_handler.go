package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ChannelHandler handles channel CRUD requests.
type ChannelHandler struct {
	repo *database.ChannelRepo
}

// NewChannelHandler creates a ChannelHandler backed by the given repo.
func NewChannelHandler(channelRepo *database.ChannelRepo) *ChannelHandler {
	return &ChannelHandler{repo: channelRepo}
}

// createChannelRequest is the JSON body for POST /api/channels.
type createChannelRequest struct {
	Name             string `json:"name"`
	YoutubeChannelID string `json:"youtube_channel_id"`
}

// GetChannels lists all channels.
func (h *ChannelHandler) GetChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if channels == nil {
		channels = []database.Channel{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(channels)
}

// GetChannel returns a single channel by id path value.
func (h *ChannelHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ch, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}

	slog.Info("channel fetched", "id", id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ch)
}

// PostChannel creates a new channel.
func (h *ChannelHandler) PostChannel(w http.ResponseWriter, r *http.Request) {
	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	id, err := h.repo.Create(r.Context(), req.Name, req.YoutubeChannelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ch, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("channel created", "id", id, "name", req.Name, "youtube_channel_id", req.YoutubeChannelID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ch)
}

// DeleteChannel deletes a channel by id path value.
func (h *ChannelHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
