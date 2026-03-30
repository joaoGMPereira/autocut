package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/config"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// NewRouter creates the HTTP router with all routes registered.
// It preserves GET /health and adds all API endpoints from the Wave-2 handlers.
func NewRouter(
	cfg *config.Config,
	db *sql.DB,
	log *slog.Logger,
	h *hub.SSEHub,
	download *handlers.DownloadHandler,
	processorH *handlers.ProcessorHandler,
	transcriptH *handlers.TranscriptHandler,
	aiH *handlers.AIHandler,
	thumbnailH *handlers.ThumbnailHandler,
	uploadH *handlers.UploadHandler,
	setupH *handlers.SetupHandler,
) http.Handler {
	mux := http.NewServeMux()

	// ── Health ──────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /health", handleHealth)

	// ── Download ────────────────────────────────────────────────────────────
	mux.HandleFunc("POST /api/download", download.PostDownload)
	mux.HandleFunc("GET /api/download/{id}/stream", download.GetDownloadStream)

	// ── Processor ───────────────────────────────────────────────────────────
	mux.HandleFunc("POST /api/cut", processorH.PostCut)
	mux.HandleFunc("GET /api/cut/{id}/stream", processorH.GetCutStream)

	mux.HandleFunc("POST /api/shorts", processorH.PostShorts)
	mux.HandleFunc("GET /api/shorts/{id}/stream", processorH.GetShortsStream)

	mux.HandleFunc("POST /api/optimize", processorH.PostOptimize)
	mux.HandleFunc("GET /api/optimize/{id}/stream", processorH.GetOptimizeStream)

	// ── Transcript ──────────────────────────────────────────────────────────
	mux.HandleFunc("POST /api/transcript", transcriptH.PostTranscript)
	mux.HandleFunc("GET /api/transcript/{id}/stream", transcriptH.GetTranscriptStream)

	// ── AI Analysis ─────────────────────────────────────────────────────────
	mux.HandleFunc("POST /api/analyze", aiH.PostAnalyze)
	mux.HandleFunc("GET /api/analyze/{id}/stream", aiH.GetAnalyzeStream)

	// ── Thumbnail ───────────────────────────────────────────────────────────
	mux.HandleFunc("POST /api/thumbnail", thumbnailH.PostThumbnail)

	// ── Upload ──────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/upload", uploadH.GetUploads)
	mux.HandleFunc("POST /api/upload", uploadH.PostUpload)
	mux.HandleFunc("GET /api/upload/{id}/stream", uploadH.GetUploadStream)

	// ── Channels ────────────────────────────────────────────────────────────
	channelH := handlers.NewChannelHandler(db)
	mux.HandleFunc("GET /api/channels", channelH.GetChannels)
	mux.HandleFunc("POST /api/channels", channelH.PostChannel)
	mux.HandleFunc("DELETE /api/channels/{id}", channelH.DeleteChannel)

	// ── Settings ────────────────────────────────────────────────────────────
	settingsH := handlers.NewSettingsHandler(db)
	mux.HandleFunc("GET /api/settings", settingsH.GetSettings)
	mux.HandleFunc("PUT /api/settings", settingsH.PutSettings)

	// ── Setup ────────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/setup/status", setupH.GetStatus)
	mux.HandleFunc("POST /api/setup/install/{tool}", setupH.PostInstall)
	mux.HandleFunc("GET /api/setup/install/{tool}/stream", setupH.GetInstallStream)
	mux.HandleFunc("GET /api/setup/dir", setupH.GetDir)

	return corsMiddleware(mux)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// devOrigins are the allowed CORS origins for local development.
// Electron loads the renderer via 127.0.0.1 while browsers use localhost —
// both must be allowed because they are treated as distinct origins by CORS.
var devOrigins = map[string]bool{
	"http://localhost:3201":   true,
	"http://127.0.0.1:3201":  true,
}

func corsMiddleware(next http.Handler) http.Handler {
	explicit := os.Getenv("CORS_ORIGIN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowed := false
		if explicit != "" {
			allowed = origin == explicit
		} else {
			allowed = devOrigins[origin]
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
