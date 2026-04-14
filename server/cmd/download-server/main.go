// Minimal HTTP server for testing the download endpoint.
// Run: cd AutoCut/server && go run ./cmd/download-server -port 4071
package main

import (
	"flag"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

func main() {
	port := flag.String("port", "4071", "HTTP port")
	flag.Parse()

	h := hub.New()
	ytDl := downloader.NewYouTubeDownloader("")
	twDl := downloader.NewTwitchDownloader("")
	dl := handlers.NewDownloadHandler(h, ytDl, twDl, nil)

	mux := http.NewServeMux()

	// CORS middleware for local dev
	withCORS := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}

	mux.HandleFunc("POST /api/download", withCORS(dl.PostDownload))
	mux.HandleFunc("OPTIONS /api/download", withCORS(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	mux.HandleFunc("GET /api/download/{id}/stream", withCORS(dl.GetDownloadStream))
	mux.HandleFunc("OPTIONS /api/download/{id}/stream", withCORS(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	slog.Info("download server listening", "port", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		slog.Error("server failed", "err", err)
	}
}
