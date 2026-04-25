package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/api"
	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/crypto"
	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/logsink"
	internaloauth "github.com/joaoGMPereira/autocut/server/internal/oauth"
	"github.com/joaoGMPereira/autocut/server/internal/pipeline"
	"github.com/joaoGMPereira/autocut/server/internal/stats"
	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

func main() {
	// CLI flags (Makefile interface)
	dir := flag.String("dir", defaultDataDir(), "Data directory (DB + downloads)")
	host := flag.String("host", "127.0.0.1", "Host to listen on")
	port := flag.String("port", "4070", "Port to listen on")
	assetsDir := flag.String("assets-dir", "", "Bundled assets directory (music, overlays)")
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

	sink := logsink.New(os.Stderr)
	slog.SetDefault(slog.New(sink))

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		slog.Error("failed to create data directory", "dir", *dir, "err", err)
		os.Exit(1)
	}

	// Augment PATH early — before any exec.LookPath or exec.Command call — so
	// tools installed in ~/.autocut/bin, /opt/homebrew/bin, or /usr/local/bin
	// are found even when the server is spawned from Electron with a restricted PATH.
	autocutDirEarly := configurator.NewAutoCutDirFromPath(*dir)
	configurator.AugmentPATH(autocutDirEarly.BinDir)

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

	// Seed asset library paths from bundled assets on first boot.
	settingRepoEarly := database.NewAppSettingRepo(db, slog.Default())
	if *assetsDir != "" {
		seedAssetPaths(settingRepoEarly, *assetsDir)
	}

	sseHub := hub.New()
	repo := database.NewPipelineRunRepo(db, slog.Default())
	historyRepo := database.NewURLHistoryRepo(db, slog.Default())
	settingRepo := database.NewAppSettingRepo(db, slog.Default())
	channelCfgRepo := database.NewChannelConfigRepo(db, slog.Default())
	channelAnalyticsRepo := database.NewChannelAnalyticsRepo(db)
	ytDlpBin := configurator.FindBinary("yt-dlp", autocutDirEarly.BinDir)
	if ytDlpBin == "" {
		ytDlpBin = "yt-dlp"
		slog.Warn("yt-dlp not found in known paths, falling back to bare name")
	}
	ytDl := downloader.NewYouTubeDownloader(ytDlpBin)
	twDl := downloader.NewTwitchDownloader(ytDlpBin)

	highlightRepo := database.NewPipelineHighlightRepo(db, slog.Default())
	clipRepo := database.NewPipelineClipRepo(db, slog.Default())
	uploadRepo := database.NewUploadRepo(db, slog.Default())
	queueStorage := uploader.NewQueueStorage(*dir)
	pipelineSvc := pipeline.NewService(repo, historyRepo, highlightRepo, clipRepo, channelCfgRepo, sseHub, ytDl, twDl, *dir, settingRepo, uploadRepo, queueStorage)
	pipelineHandler := handlers.NewPipelineHandler(pipelineSvc, sseHub)
	downloadHandler := handlers.NewDownloadHandler(sseHub, ytDl, twDl, nil)
	urlHistoryHandler := handlers.NewURLHistoryHandler(historyRepo)

	autocutDir := autocutDirEarly
	if err := autocutDir.Ensure(); err != nil {
		slog.Warn("failed to ensure autocut dirs", "err", err)
	}
	cfg := configurator.New(autocutDir, settingRepo)
	setupHandler := handlers.NewSetupHandler(sseHub, cfg)
	settingsHandler := handlers.NewSettingsHandler(settingRepo)

	cipher, err := crypto.LoadOrCreate(filepath.Join(*dir, "db.key"))
	if err != nil {
		slog.Error("failed to load token cipher", "err", err)
		os.Exit(1)
	}
	channelRepo := database.NewChannelRepo(db, slog.Default(), cipher)
	oauthSecretRepo := database.NewOAuthClientSecretRepo(db, slog.Default())
	sessionMgr := internaloauth.NewSessionManager(channelRepo)
	channelHandler := handlers.NewChannelHandler(channelRepo, *dir)
	oauthHandler := handlers.NewOAuthHandler(db, channelRepo, sessionMgr)
	ollamaHandler := handlers.NewOllamaHandler()
	statsHandler := handlers.NewStatsHandler(stats.New())
	mediaLibraryHandler := handlers.NewMediaLibraryHandler(settingRepo)
	previewHandler := handlers.NewPreviewHandler(repo, channelCfgRepo, settingRepo, *dir, sseHub)
	thumbnailHandler := handlers.NewThumbnailHandler(repo, clipRepo, settingRepo, sseHub, *dir)
	queueHandler := handlers.NewQueueHandler(uploadRepo, *dir)

	// Claude CLI + metadata generator
	claudeCLI, claudeErr := ai.NewClaudeCLI()
	if claudeErr != nil {
		slog.Warn("claude CLI not available — metadata generation disabled", "err", claudeErr)
	}
	channelAnalyzer := ai.NewChannelAnalyzer(ytDlpBin, channelAnalyticsRepo)
	metadataGen := ai.NewMetadataGenerator(claudeCLI, repo, clipRepo, highlightRepo, channelCfgRepo, settingRepo, sseHub, channelAnalyticsRepo, channelAnalyzer)
	metadataGen.SetChannelBaseRepo(channelRepo)
	pipelineSvc.SetMetadataGenerator(metadataGen)
	metadataHandler := handlers.NewMetadataHandler(repo, clipRepo, metadataGen, sseHub)
	channelConfigHandler := handlers.NewChannelConfigHandler(channelCfgRepo)
	logsHandler := handlers.NewLogsHandler(sink)

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
		previewHandler,
		thumbnailHandler,
		metadataHandler,
		queueHandler,
		channelConfigHandler,
		logsHandler,
	)

	// Start YouTube upload worker (uploads scheduled items when publish_at is reached)
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()
	ytUploader := uploader.NewYouTubeUploader(channelRepo, oauthSecretRepo, uploadRepo)
	uploadWorker := uploader.NewWorker(ytUploader, uploadRepo)
	go uploadWorker.Start(serverCtx)
	queueHandler.SetUploadTrigger(func() { uploadWorker.Trigger(serverCtx) })

	addr := *host + ":" + *port
	slog.Info("server starting", "addr", addr, "data_dir", *dir)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

// seedAssetPaths seeds music_library_path and overlay_library_path in app_settings
// from the bundled assets directory if they are not yet set.
func seedAssetPaths(repo *database.AppSettingRepo, assetsDir string) {
	ctx := context.Background()
	type assetSeed struct {
		key  string
		path string
	}
	seeds := []assetSeed{
		{"music_library_path", filepath.Join(assetsDir, "shared", "music")},
		{"overlay_library_path", filepath.Join(assetsDir, "shared", "overlays")},
	}
	for _, s := range seeds {
		current, _ := repo.Get(ctx, s.key)
		if current != "" {
			continue // user already configured, don't overwrite
		}
		info, err := os.Stat(s.path)
		if err != nil || !info.IsDir() {
			slog.Warn("bundled asset dir not found, skipping", "key", s.key, "path", s.path)
			continue
		}
		if err := repo.Set(ctx, s.key, s.path); err != nil {
			slog.Warn("failed to seed asset path", "key", s.key, "err", err)
		} else {
			slog.Info("seeded asset path", "key", s.key, "path", s.path)
		}
	}
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".autocut"
	}
	return filepath.Join(home, ".autocut")
}
