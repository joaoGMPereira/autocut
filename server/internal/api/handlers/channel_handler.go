package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ChannelHandler handles channel CRUD requests.
type ChannelHandler struct {
	repo    *database.ChannelRepo
	dataDir string
}

// NewChannelHandler creates a ChannelHandler backed by the given repo.
func NewChannelHandler(channelRepo *database.ChannelRepo, dataDir string) *ChannelHandler {
	return &ChannelHandler{repo: channelRepo, dataDir: dataDir}
}

// createChannelRequest is the JSON body for POST /api/channels.
type createChannelRequest struct {
	Name             string `json:"name"`
	YoutubeChannelID string `json:"youtube_channel_id"`
}

// avatarPath returns the local cache path for a channel avatar.
func (h *ChannelHandler) avatarPath(channelID int64) string {
	return filepath.Join(h.dataDir, "avatars", fmt.Sprintf("%d.jpg", channelID))
}

// downloadAvatar fetches the avatar from remoteURL and saves to local cache.
func (h *ChannelHandler) downloadAvatar(ctx context.Context, channelID int64, remoteURL string) error {
	if remoteURL == "" {
		return nil
	}
	dest := h.avatarPath(channelID)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	// Skip if already cached
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("avatar fetch status %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// GetChannels lists all channels and prefetches avatars in the background.
func (h *ChannelHandler) GetChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if channels == nil {
		channels = []database.Channel{}
	}

	// Prefetch missing avatars in the background
	go func() {
		for _, ch := range channels {
			if ch.AvatarURL == "" {
				continue
			}
			if _, statErr := os.Stat(h.avatarPath(ch.ID)); statErr == nil {
				continue // already cached
			}
			if dlErr := h.downloadAvatar(context.Background(), ch.ID, ch.AvatarURL); dlErr != nil {
				slog.Warn("avatar prefetch failed", "channel_id", ch.ID, "err", dlErr)
			} else {
				slog.Info("avatar cached", "channel_id", ch.ID)
			}
		}
	}()

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

// GetAvatar serves the cached channel avatar, downloading it on first request.
func (h *ChannelHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dest := h.avatarPath(id)

	// Cache miss — fetch channel and download avatar
	if _, statErr := os.Stat(dest); os.IsNotExist(statErr) {
		ch, chErr := h.repo.GetByID(r.Context(), id)
		if chErr != nil {
			http.Error(w, "channel not found", http.StatusNotFound)
			return
		}
		if ch.AvatarURL == "" {
			http.NotFound(w, r)
			return
		}
		if dlErr := h.downloadAvatar(r.Context(), id, ch.AvatarURL); dlErr != nil {
			slog.Warn("avatar download failed", "channel_id", id, "err", dlErr)
			http.Error(w, "failed to fetch avatar", http.StatusBadGateway)
			return
		}
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, dest)
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

	// Clean up cached avatar
	_ = os.Remove(h.avatarPath(id))

	w.WriteHeader(http.StatusNoContent)
}
