package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joaoGMPereira/autocut/server/internal/api"
	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/crypto"
	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	internaloauth "github.com/joaoGMPereira/autocut/server/internal/oauth"
	"github.com/joaoGMPereira/autocut/server/internal/pipeline"
	"github.com/joaoGMPereira/autocut/server/internal/stats"
)

func main() {
	// CLI flags (Makefile interface)
	dir := flag.String("dir", defaultDataDir(), "Data directory (DB + downloads)")
	host := flag.String("host", "127.0.0.1", "Host to listen on")
	port := flag.String("port", "4070", "Port to listen on")
	migrateOnly := flag.Bool("migrate-only", false, "Run DB migrations and exit")
	flag.Parse()

	// Env overrides for flags not already set via CLI (convenience for local dev)
	if *dir == defaultDataDir() {
		if v := os.Getenv("DATA_DIR"); v != "" {
			*dir = v
		}
	}
	if *port == "4070" {
		if v := os.Getenv("PORT"); v != "" {
			*port = v
		}
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		slog.Error("failed to create data directory", "dir", *dir, "err", err)
		os.Exit(1)
	}

	db, err := database.Open(*dir, slog.Default())
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// -migrate-only: used by `make migrate` — run migrations and exit cleanly.
	if *migrateOnly {
		slog.Info("migrations complete", "dir", *dir)
		return
	}

	sseHub := hub.New()
	repo := database.NewPipelineRunRepo(db, slog.Default())
	historyRepo := database.NewURLHistoryRepo(db, slog.Default())
	settingRepo := database.NewAppSettingRepo(db, slog.Default())
	ytDl := downloader.NewYouTubeDownloader("")
	twDl := downloader.NewTwitchDownloader("")

	pipelineSvc := pipeline.NewService(repo, historyRepo, sseHub, ytDl, twDl, *dir)
	pipelineHandler := handlers.NewPipelineHandler(pipelineSvc, sseHub)
	downloadHandler := handlers.NewDownloadHandler(sseHub, ytDl, twDl, nil)
	urlHistoryHandler := handlers.NewURLHistoryHandler(historyRepo)

	autocutDir := configurator.NewAutoCutDirFromPath(*dir)
	if err := autocutDir.Ensure(); err != nil {
		slog.Warn("failed to ensure autocut dirs", "err", err)
	}
	cfg := configurator.New(autocutDir)
	setupHandler := handlers.NewSetupHandler(sseHub, cfg)
	settingsHandler := handlers.NewSettingsHandler(settingRepo)

	cipher, err := crypto.LoadOrCreate(filepath.Join(*dir, "db.key"))
	if err != nil {
		slog.Error("failed to load token cipher", "err", err)
		os.Exit(1)
	}
	channelRepo := database.NewChannelRepo(db, slog.Default(), cipher)
	sessionMgr := internaloauth.NewSessionManager(channelRepo)
	channelHandler := handlers.NewChannelHandler(channelRepo)
	oauthHandler := handlers.NewOAuthHandler(db, channelRepo, sessionMgr)
	ollamaHandler := handlers.NewOllamaHandler()
	statsHandler := handlers.NewStatsHandler(stats.New())
	mediaLibraryHandler := handlers.NewMediaLibraryHandler(settingRepo)

	router := api.NewRouter(
		pipelineHandler,
		downloadHandler,
		setupHandler,
		statsHandler,
		urlHistoryHandler,
		settingsHandler,
		channelHandler,
		oauthHandler,
		ollamaHandler,
		mediaLibraryHandler,
	)

	addr := *host + ":" + *port
	slog.Info("server starting", "addr", addr, "data_dir", *dir)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".autocut"
	}
	return filepath.Join(home, ".autocut")
}
