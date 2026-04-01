package handlers

import (
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/downloader"
)

// MetadataHandler handles video metadata requests.
type MetadataHandler struct {
	downloader *downloader.YouTubeDownloader
	log        *slog.Logger
}

// NewMetadataHandler creates a MetadataHandler.
func NewMetadataHandler(dl *downloader.YouTubeDownloader) *MetadataHandler {
	return &MetadataHandler{
		downloader: dl,
		log:        slog.With("component", "api", "handler", "metadata"),
	}
}

// GetMetadata handles GET /api/metadata?url={url}.
func (h *MetadataHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "url query parameter is required")
		return
	}

	info, err := h.downloader.ExtractMetadata(url)
	if err != nil {
		h.log.Error("metadata extraction failed", "err", err, "url", url)
		writeError(w, http.StatusInternalServerError, "metadata_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"video_id":      info.VideoID,
		"title":         info.Title,
		"description":   info.Description,
		"thumbnail_url": info.ThumbnailURL,
		"duration":      info.Duration.Seconds(),
	})
}

