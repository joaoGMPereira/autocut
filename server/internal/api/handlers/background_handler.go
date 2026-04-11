package handlers

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// BackgroundHandler handles CRUD operations for per-channel thumbnail backgrounds.
// Backgrounds are stored on disk under {dataDir}/backgrounds/{channelID}/{uuid}{ext}
// and registered in the thumbnail_backgrounds table.
type BackgroundHandler struct {
	repo    *database.ThumbnailBackgroundRepo
	log     *slog.Logger
	dataDir string
}

// NewBackgroundHandler creates a BackgroundHandler.
// dataDir is the root data directory (e.g. ~/.autocut).
func NewBackgroundHandler(db *sql.DB, log *slog.Logger, dataDir string) *BackgroundHandler {
	return &BackgroundHandler{
		repo:    database.NewThumbnailBackgroundRepo(db, log),
		log:     log.With("component", "api", "handler", "background"),
		dataDir: dataDir,
	}
}

// backgroundResponse is the public JSON view of a ThumbnailBackground.
// It includes both the raw file_path (absolute server path) and a browser-accessible url.
type backgroundResponse struct {
	ID        int64  `json:"id"`
	ChannelID int64  `json:"channel_id"`
	Name      string `json:"name"`
	FilePath  string `json:"file_path"`
	URL       string `json:"url"`
	IsDefault bool   `json:"is_default"`
	CreatedAt int64  `json:"created_at"`
}

// ListBackgrounds handles GET /api/channels/{id}/backgrounds.
// Returns all background records for the given channel as a JSON array.
func (h *BackgroundHandler) ListBackgrounds(w http.ResponseWriter, r *http.Request) {
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

	bgs, err := h.repo.ListByChannel(r.Context(), channelID)
	if err != nil {
		h.log.Error("list backgrounds failed", "channelID", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	resp := make([]backgroundResponse, len(bgs))
	for i, bg := range bgs {
		resp[i] = backgroundResponse{
			ID:        bg.ID,
			ChannelID: bg.ChannelID,
			Name:      bg.Name,
			FilePath:  bg.FilePath,
			URL:       "/files/backgrounds/" + strconv.Itoa(int(channelID)) + "/" + filepath.Base(bg.FilePath),
			IsDefault: bg.IsDefault,
			CreatedAt: bg.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// UploadBackground handles POST /api/channels/{id}/backgrounds.
// Accepts multipart/form-data with field "file" (the image) and optional "name".
// Saves the file to {dataDir}/backgrounds/{channelID}/{uuid}{ext} and creates a DB record.
func (h *BackgroundHandler) UploadBackground(w http.ResponseWriter, r *http.Request) {
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

	// 32 MB max in-memory, rest spills to disk.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "parse_form_failed", err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "form field 'file' is required")
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	// Preserve original extension; use a UUID for the file name.
	ext := filepath.Ext(header.Filename)
	uuid := newJobID()
	fileName := uuid + ext

	destDir := filepath.Join(h.dataDir, "backgrounds", strconv.FormatInt(channelID, 10))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		h.log.Error("mkdir failed", "channelID", channelID, "dir", destDir, "err", err)
		writeError(w, http.StatusInternalServerError, "storage_failed", err.Error())
		return
	}

	destPath := filepath.Join(destDir, fileName)
	dst, err := os.Create(destPath)
	if err != nil {
		h.log.Error("create file failed", "channelID", channelID, "path", destPath, "err", err)
		writeError(w, http.StatusInternalServerError, "storage_failed", err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.log.Error("copy file failed", "channelID", channelID, "path", destPath, "err", err)
		writeError(w, http.StatusInternalServerError, "storage_failed", err.Error())
		return
	}

	id, err := h.repo.Create(r.Context(), &database.ThumbnailBackground{
		ChannelID: channelID,
		Name:      name,
		FilePath:  destPath,
		IsDefault: false,
	})
	if err != nil {
		h.log.Error("create background record failed", "channelID", channelID, "err", err)
		// Best-effort cleanup of the file we just saved.
		if removeErr := os.Remove(destPath); removeErr != nil {
			h.log.Warn("cleanup failed after db error", "path", destPath, "err", removeErr)
		}
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	bg, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("fetch background after create failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}

	h.log.Info("background uploaded", "channelID", channelID, "bgID", id, "name", name)
	writeJSON(w, http.StatusCreated, bg)
}

// DeleteBackground handles DELETE /api/channels/{id}/backgrounds/{bg_id}.
// Verifies the background belongs to the channel, removes the file, then removes the DB record.
func (h *BackgroundHandler) DeleteBackground(w http.ResponseWriter, r *http.Request) {
	rawChannelID := r.PathValue("id")
	rawBgID := r.PathValue("bg_id")
	if rawChannelID == "" || rawBgID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id and bg_id are required")
		return
	}

	channelID, err := strconv.ParseInt(rawChannelID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "id must be an integer")
		return
	}
	bgID, err := strconv.ParseInt(rawBgID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "bg_id must be an integer")
		return
	}

	bg, err := h.repo.GetByID(r.Context(), bgID)
	if err != nil {
		h.log.Error("get background failed", "bgID", bgID, "err", err)
		writeError(w, http.StatusNotFound, "not_found", "background not found")
		return
	}

	// Verify ownership — prevent cross-channel deletions.
	if bg.ChannelID != channelID {
		writeError(w, http.StatusForbidden, "forbidden", "background does not belong to this channel")
		return
	}

	// Remove the file. A missing file is not fatal — the record still needs cleanup.
	if err := os.Remove(bg.FilePath); err != nil && !os.IsNotExist(err) {
		h.log.Warn("remove background file failed (non-fatal)", "path", bg.FilePath, "err", err)
	}

	if err := h.repo.Delete(r.Context(), bgID); err != nil {
		h.log.Error("delete background record failed", "bgID", bgID, "err", err)
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	h.log.Info("background deleted", "channelID", channelID, "bgID", bgID)
	w.WriteHeader(http.StatusNoContent)
}
