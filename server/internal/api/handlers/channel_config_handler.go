package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ChannelConfigHandler handles channel configuration and music blacklist endpoints.
type ChannelConfigHandler struct {
	repo          *database.ChannelConfigRepo
	blacklistRepo *database.MusicBlacklistRepo
	dataDir       string
	log           *slog.Logger
}

// NewChannelConfigHandler creates a ChannelConfigHandler.
// dataDir is used to determine where logo files are saved (dataDir/logos/{channel_id}/).
func NewChannelConfigHandler(
	repo *database.ChannelConfigRepo,
	blacklistRepo *database.MusicBlacklistRepo,
	dataDir string,
) *ChannelConfigHandler {
	return &ChannelConfigHandler{
		repo:          repo,
		blacklistRepo: blacklistRepo,
		dataDir:       dataDir,
		log:           slog.With("component", "api", "handler", "channel_config"),
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
		h.log.Error("get channel config failed", "component", "api", "op", "get_config", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig handles PUT /api/channels/{id}/config.
// Accepts a partial JSON body — only provided fields are applied.
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

	// Load existing config first so unset fields keep their current values.
	existing, err := h.repo.GetByChannelID(r.Context(), channelID)
	if err != nil {
		h.log.Error("get channel config for update failed", "component", "api", "op", "update_config", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	if err := json.NewDecoder(r.Body).Decode(existing); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	existing.ChannelID = channelID

	if err := h.repo.Update(r.Context(), existing); err != nil {
		h.log.Error("update channel config failed", "component", "api", "op", "update_config", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	h.log.Info("channel config updated", "channel_id", channelID)
	w.WriteHeader(http.StatusOK)
}

// PostLogo handles POST /api/channels/{id}/config/logo.
// Expects multipart/form-data with a "logo" field.
// Saves file to {dataDir}/logos/{channel_id}/logo.{ext} and updates branding_logo_path.
func (h *ChannelConfigHandler) PostLogo(w http.ResponseWriter, r *http.Request) {
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

	const maxSize = 10 << 20 // 10 MB
	if err := r.ParseMultipartForm(maxSize); err != nil {
		writeError(w, http.StatusBadRequest, "multipart_error", "failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		h.log.Error("form file missing", "component", "api", "op", "post_logo", "channel_id", channelID, "err", err)
		writeError(w, http.StatusBadRequest, "missing_file", "logo file is required")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isAllowedLogoExt(ext) {
		ext = ".png"
	}

	logoDir := filepath.Join(h.dataDir, "logos", strconv.FormatInt(channelID, 10))
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		h.log.Error("create logo dir failed", "component", "api", "op", "post_logo", "channel_id", channelID, "dir", logoDir, "err", err)
		writeError(w, http.StatusInternalServerError, "dir_error", "failed to create logo directory")
		return
	}

	logoPath := filepath.Join(logoDir, "logo"+ext)
	dst, err := os.Create(logoPath)
	if err != nil {
		h.log.Error("create logo file failed", "component", "api", "op", "post_logo", "channel_id", channelID, "path", logoPath, "err", err)
		writeError(w, http.StatusInternalServerError, "file_error", "failed to create logo file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.log.Error("write logo file failed", "component", "api", "op", "post_logo", "channel_id", channelID, "path", logoPath, "err", err)
		writeError(w, http.StatusInternalServerError, "write_error", "failed to write logo file")
		return
	}

	// Update branding_logo_path in channel_config.
	cfg, err := h.repo.GetByChannelID(r.Context(), channelID)
	if err != nil {
		h.log.Error("get config for logo update failed", "component", "api", "op", "post_logo", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	cfg.BrandingLogoEnabled = true
	cfg.BrandingLogoPath.Valid = true
	cfg.BrandingLogoPath.String = logoPath

	if err := h.repo.Update(r.Context(), cfg); err != nil {
		h.log.Error("update logo path failed", "component", "api", "op", "post_logo", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	h.log.Info("logo saved", "component", "api", "op", "post_logo", "channel_id", channelID, "path", logoPath)
	writeJSON(w, http.StatusOK, map[string]string{"logo_path": logoPath})
}

// GetMusicBlacklist handles GET /api/channels/{id}/music-blacklist.
func (h *ChannelConfigHandler) GetMusicBlacklist(w http.ResponseWriter, r *http.Request) {
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

	entries, err := h.blacklistRepo.ListByChannel(r.Context(), channelID)
	if err != nil {
		h.log.Error("list music blacklist failed", "component", "api", "op", "get_blacklist", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	if entries == nil {
		entries = []database.MusicBlacklistEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// PostMusicBlacklist handles POST /api/channels/{id}/music-blacklist.
// Body: {"pattern": "string"}
func (h *ChannelConfigHandler) PostMusicBlacklist(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Pattern string `json:"pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if strings.TrimSpace(req.Pattern) == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "pattern is required")
		return
	}

	entry, err := h.blacklistRepo.Create(r.Context(), channelID, req.Pattern)
	if err != nil {
		h.log.Error("create music blacklist entry failed", "component", "api", "op", "post_blacklist", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	h.log.Info("music blacklist entry created", "component", "api", "op", "post_blacklist", "channel_id", channelID, "id", entry.ID, "pattern", entry.Pattern)
	writeJSON(w, http.StatusCreated, entry)
}

// DeleteMusicBlacklist handles DELETE /api/channels/{id}/music-blacklist/{pattern_id}.
func (h *ChannelConfigHandler) DeleteMusicBlacklist(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	if rawID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}

	rawPatternID := r.PathValue("pattern_id")
	if rawPatternID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "pattern_id is required")
		return
	}

	channelID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "id must be an integer")
		return
	}

	patternID, err := strconv.ParseInt(rawPatternID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "pattern_id must be an integer")
		return
	}

	if err := h.blacklistRepo.Delete(r.Context(), patternID); err != nil {
		h.log.Error("delete music blacklist entry failed", "component", "api", "op", "delete_blacklist", "channel_id", channelID, "pattern_id", patternID, "err", err)
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	h.log.Info("music blacklist entry deleted", "component", "api", "op", "delete_blacklist", "channel_id", channelID, "pattern_id", patternID)
	w.WriteHeader(http.StatusNoContent)
}

// isAllowedLogoExt returns true if the file extension is an accepted image format.
func isAllowedLogoExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	}
	return false
}
