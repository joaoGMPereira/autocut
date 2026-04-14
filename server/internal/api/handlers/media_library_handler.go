package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// audioExtensions is the set of file extensions treated as audio.
var audioExtensions = map[string]bool{
	".mp3": true, ".m4a": true, ".aac": true,
	".wav": true, ".flac": true, ".ogg": true,
}

// videoExtensions is the set of file extensions treated as video overlays.
var videoExtensions = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true,
	".avi": true, ".webm": true,
}

// MediaLibraryHandler serves directory listings for media assets.
type MediaLibraryHandler struct {
	settingRepo AppSettingRepoFace
}

// NewMediaLibraryHandler creates a MediaLibraryHandler.
func NewMediaLibraryHandler(repo AppSettingRepoFace) *MediaLibraryHandler {
	return &MediaLibraryHandler{settingRepo: repo}
}

// GetMusicLibrary lists audio files in the directory configured via the
// `music_library_path` app setting.
// GET /api/music-library
func (h *MediaLibraryHandler) GetMusicLibrary(w http.ResponseWriter, r *http.Request) {
	h.listDir(w, r, "music_library_path", audioExtensions)
}

// GetOverlayLibrary lists video files in the directory configured via the
// `overlay_library_path` app setting.
// GET /api/overlay-library
func (h *MediaLibraryHandler) GetOverlayLibrary(w http.ResponseWriter, r *http.Request) {
	h.listDir(w, r, "overlay_library_path", videoExtensions)
}

func (h *MediaLibraryHandler) listDir(
	w http.ResponseWriter,
	r *http.Request,
	settingKey string,
	exts map[string]bool,
) {
	// Directory path comes exclusively from the trusted app settings database.
	// Client-supplied paths are intentionally not accepted to prevent directory traversal.
	settings, err := h.settingRepo.List(r.Context())
	if err != nil {
		slog.Error("list settings failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var dir string
	for _, s := range settings {
		if s.Key == settingKey {
			dir = s.Value
			break
		}
	}

	if dir == "" {
		// No directory configured — return empty list.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]string{})
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Warn("media library dir unreadable", "dir", dir, "err", err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]string{})
		return
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if exts[ext] {
			files = append(files, e.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}
