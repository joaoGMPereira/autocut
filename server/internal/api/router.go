package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
)

// NewRouter builds and returns the application HTTP handler.
// All routes are registered on a stdlib ServeMux with CORS and request logging middleware.
func NewRouter(
	ph *handlers.PipelineHandler,
	dh *handlers.DownloadHandler,
	sh *handlers.SetupHandler,
	sth *handlers.StatsHandler,
	uhh *handlers.URLHistoryHandler,
	seth *handlers.SettingsHandler,
	ch *handlers.ChannelHandler,
	oh *handlers.OAuthHandler,
	olh *handlers.OllamaHandler,
	mlh *handlers.MediaLibraryHandler,
	pvh *handlers.PreviewHandler,
	th *handlers.ThumbnailHandler,
	mh *handlers.MetadataHandler,
	qh *handlers.QueueHandler,
	cch *handlers.ChannelConfigHandler,
	lh *handlers.LogsHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Pipeline run endpoints (consumed by pipelineStore.ts)
	mux.HandleFunc("POST /api/pipeline/runs", ph.PostRuns)
	mux.HandleFunc("GET /api/pipeline/runs/prior-clips", ph.GetPriorClips)
	mux.HandleFunc("GET /api/pipeline/runs/{id}", ph.GetRun)
	mux.HandleFunc("GET /api/pipeline/runs/{id}/stream", ph.GetRunStream)
	mux.HandleFunc("POST /api/pipeline/runs/{id}/advance", ph.PostAdvance)
	mux.HandleFunc("GET /api/pipeline/runs/{id}/clips", ph.GetRunClips)
	mux.HandleFunc("POST /api/pipeline/runs/{id}/cancel", ph.PostCancel)
	mux.HandleFunc("POST /api/pipeline/runs/{id}/redownload", ph.PostRedownload)

	// URL history endpoints (consumed by urlHistoryStore.ts)
	mux.HandleFunc("GET /api/url-history", uhh.GetHistory)
	mux.HandleFunc("DELETE /api/url-history/{id}", uhh.DeleteEntry)
	mux.HandleFunc("DELETE /api/url-history", uhh.DeleteAll)

	// Download endpoints (used by /download-test and direct download calls)
	mux.HandleFunc("POST /api/download", dh.PostDownload)
	mux.HandleFunc("GET /api/download/{id}/stream", dh.GetDownloadStream)

	// Setup endpoints — tool status + install + whisper models
	mux.HandleFunc("GET /api/setup/status", sh.GetStatus)
	mux.HandleFunc("POST /api/setup/sync", sh.PostSync)
	mux.HandleFunc("GET /api/setup/dir", sh.GetDir)
	mux.HandleFunc("POST /api/setup/install/{tool}", sh.PostInstall)
	mux.HandleFunc("GET /api/setup/install/{tool}/stream", sh.GetInstallStream)
	mux.HandleFunc("GET /api/setup/check-update/{tool}", sh.GetCheckUpdate)
	mux.HandleFunc("GET /api/setup/whisper/models", sh.GetWhisperModels)
	mux.HandleFunc("POST /api/setup/whisper/models/{model}", sh.PostWhisperModel)
	mux.HandleFunc("GET /api/setup/whisper/models/{model}/stream", sh.GetWhisperModelStream)
	mux.HandleFunc("GET /api/setup/hardware", sh.GetHardware)
	mux.HandleFunc("POST /api/setup/tools/{tool}/path", sh.PostCustomPath)
	mux.HandleFunc("DELETE /api/setup/tools/{tool}/path", sh.DeleteCustomPath)

	// App settings endpoints
	mux.HandleFunc("GET /api/settings", seth.GetSettings)
	mux.HandleFunc("PUT /api/settings", seth.PutSetting)
	mux.HandleFunc("DELETE /api/settings", seth.DeleteSetting)

	// Channel endpoints
	mux.HandleFunc("GET /api/channels", ch.GetChannels)
	mux.HandleFunc("POST /api/channels", ch.PostChannel)
	mux.HandleFunc("GET /api/channels/{id}", ch.GetChannel)
	mux.HandleFunc("DELETE /api/channels/{id}", ch.DeleteChannel)
	mux.HandleFunc("GET /api/channels/{id}/avatar", ch.GetAvatar)
	mux.HandleFunc("GET /api/channels/{id}/config", cch.GetConfig)
	mux.HandleFunc("POST /api/channels/{id}/auth", oh.InitOAuthFlow)
	mux.HandleFunc("POST /api/channels/{id}/auth/refresh", oh.RefreshToken)

	// OAuth profile endpoints
	mux.HandleFunc("GET /api/oauth/profiles", oh.GetOAuthProfiles)
	mux.HandleFunc("POST /api/oauth/profiles", oh.PostOAuthProfile)
	mux.HandleFunc("PATCH /api/oauth/profiles/{id}", oh.PatchOAuthProfile)
	mux.HandleFunc("DELETE /api/oauth/profiles/{id}", oh.DeleteOAuthProfile)
	mux.HandleFunc("POST /api/oauth/profiles/{id}/set-default", oh.PostSetDefaultProfile)

	// Ollama models proxy
	mux.HandleFunc("GET /api/ollama/models", olh.GetOllamaModels)

	// Media library endpoints
	mux.HandleFunc("GET /api/music-library", mlh.GetMusicLibrary)
	mux.HandleFunc("GET /api/overlay-library", mlh.GetOverlayLibrary)

	// Preview endpoints
	mux.HandleFunc("POST /api/pipeline/runs/{id}/preview", pvh.PostPreview)
	mux.HandleFunc("GET /api/pipeline/runs/{id}/preview/file", pvh.GetPreviewFile)

	// Thumbnail endpoints
	mux.HandleFunc("POST /api/thumbnail/runs/{id}/generate", th.PostBatchGenerate)
	mux.HandleFunc("POST /api/thumbnail/runs/{id}/clips/{clipId}/generate", th.PostSingleGenerate)
	mux.HandleFunc("POST /api/thumbnail/runs/{id}/clips/{clipId}/preview-text", th.PostPreviewText)
	mux.HandleFunc("GET /api/thumbnail/runs/{id}/clips/{clipId}/file", th.GetThumbnailFile)
	mux.HandleFunc("GET /api/thumbnail/fonts", th.GetFonts)
	mux.HandleFunc("GET /api/thumbnail/templates", th.GetTemplates)
	mux.HandleFunc("POST /api/thumbnail/templates", th.PostTemplate)
	mux.HandleFunc("DELETE /api/thumbnail/templates/{name}", th.DeleteTemplate)
	mux.HandleFunc("POST /api/thumbnail/runs/{id}/base-image", th.PostUploadClipImage)

	// Metadata endpoints
	mux.HandleFunc("POST /api/metadata/runs/{id}/generate", mh.PostBatchGenerate)
	mux.HandleFunc("POST /api/metadata/runs/{id}/clips/{clipId}/generate", mh.PostSingleGenerate)
	mux.HandleFunc("POST /api/pipeline/runs/{id}/gates/review-metadata", mh.PostReviewMetadata)
	mux.HandleFunc("POST /api/pipeline/runs/{id}/gates/review-clips", ph.PostReviewClips)
	mux.HandleFunc("POST /api/pipeline/runs/{id}/gates/upload-confirm", ph.PostUploadConfirm)

	// Queue endpoints
	mux.HandleFunc("GET /api/queue", qh.GetQueue)
	mux.HandleFunc("DELETE /api/queue/{id}", qh.DeleteQueue)
	mux.HandleFunc("POST /api/queue/{id}/retry", qh.PostRetry)
	mux.HandleFunc("POST /api/queue/{id}/schedule", qh.PostSchedule)
	mux.HandleFunc("POST /api/queue/bulk-schedule", qh.PostBulkSchedule)
	mux.HandleFunc("POST /api/queue/upload-now", qh.PostUploadNow)
	mux.HandleFunc("POST /api/queue/save-local", qh.PostSaveLocal)
	mux.HandleFunc("GET /api/queue/{id}/thumbnail", qh.GetThumbnail)

	// Stats endpoint
	mux.HandleFunc("GET /api/stats", sth.GetStats)

	// In-app log viewer endpoints
	mux.HandleFunc("GET /api/logs/stream", lh.GetStream)
	mux.HandleFunc("GET /api/logs/history", lh.GetHistory)

	// Health check (both paths — Electron polls /health, web uses /api/health)
	health := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /api/health", health)

	return cors(requestLog(mux))
}

// cors adds permissive CORS headers for Electron (file://) and local Next.js origins.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin returns true for Electron and localhost origins.
func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	return origin == "file://" ||
		strings.HasPrefix(origin, "http://localhost") ||
		strings.HasPrefix(origin, "http://127.0.0.1")
}

// requestLog logs each incoming request with method, path, status, and duration.
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
