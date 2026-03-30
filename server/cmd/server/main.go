package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/api"
	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/config"
	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/processor"
	"github.com/joaoGMPereira/autocut/server/internal/thumbnail"
	"github.com/joaoGMPereira/autocut/server/internal/transcript"
)

func main() {
	hostFlag := flag.String("host", "127.0.0.1", "bind host")
	portFlag := flag.Int("port", 4070, "listen port")
	dirFlag := flag.String("dir", "", "data directory")
	flag.Parse()

	dataDir := *dirFlag
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get home dir: %v\n", err)
			os.Exit(1)
		}
		dataDir = fmt.Sprintf("%s/.autocut", home)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create data dir: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Structured logging: text in dev, JSON in prod
	var slogHandler slog.Handler
	if cfg.Env == "production" {
		slogHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		slogHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	logger := slog.New(slogHandler)
	slog.SetDefault(logger)

	db, err := database.Open(dataDir, logger)
	if err != nil {
		logger.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// ── Init ~/.autocut/ directory structure ─────────────────────────────────

	acDir, err := configurator.NewAutoCutDir()
	if err != nil {
		slog.Error("failed to init autocut dir", "err", err)
		os.Exit(1)
	}
	// If config specifies a custom DataDir, use it as the AutoCut root.
	if cfg.DataDir != "" {
		acDir = configurator.NewAutoCutDirFromRoot(cfg.DataDir)
	}
	if err := acDir.Ensure(); err != nil {
		slog.Error("failed to create autocut dirs", "err", err)
		os.Exit(1)
	}

	// Init tool configurator
	toolCfg := configurator.New(acDir)
	paths := toolCfg.ResolvedPaths()
	slog.Info("tool paths resolved", "paths", paths)

	// ── Wire up dependencies ─────────────────────────────────────────────────

	h := hub.New()

	// Downloader — pass resolved bin paths from configurator
	ytDl := downloader.NewYouTubeDownloader(paths["yt-dlp"])
	twDl := downloader.NewTwitchDownloader(paths["TwitchDownloaderCLI"])
	downloadH := handlers.NewDownloadHandler(h, ytDl, twDl)

	// Processor — pass resolved ffmpeg path from configurator
	ffmpegProc := processor.NewFFmpegProcessor(paths["ffmpeg"])
	shortsGen := processor.NewShortsGenerator(paths["ffmpeg"])
	optimizer := processor.NewVideoOptimizerProcessor(paths["ffmpeg"])
	processorH := handlers.NewProcessorHandler(h, ffmpegProc, shortsGen, optimizer)

	// Transcript — pass acDir.TransCacheDir and resolved whisper path
	whisperTranscriber := transcript.New(transcript.WhisperConfig{
		BinPath: paths["whisper"],
	})
	transcriptCache := transcript.NewCache(acDir.TransCacheDir)
	transcriptH := handlers.NewTranscriptHandler(h, whisperTranscriber, transcriptCache)

	// AI
	ollamaClient := ai.NewOllamaProvider("http://localhost:11434", 5*time.Minute)
	detector := ai.NewDetector(ollamaClient, ai.DetectorConfig{})
	aiH := handlers.NewAIHandler(h, detector)

	// Thumbnail — pass resolved ffmpeg and convert (ImageMagick) paths
	thumbnailGen := thumbnail.New(paths["ffmpeg"], paths["convert"])
	thumbnailH := handlers.NewThumbnailHandler(thumbnailGen)

	// Upload — nil uploader until OAuth is configured
	// TODO: pass acDir.TokensDir when uploader.NewUploadHandler accepts it
	uploadH := handlers.NewUploadHandler(h, nil, nil, nil)

	// Setup handler — wires tool configurator for /api/setup/* routes
	setupH := handlers.NewSetupHandler(h, toolCfg)

	// ── Build router ─────────────────────────────────────────────────────────

	router := api.NewRouter(cfg, db, logger, h,
		downloadH, processorH, transcriptH, aiH, thumbnailH, uploadH, setupH)

	addr := fmt.Sprintf("%s:%d", *hostFlag, *portFlag)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // disabled for SSE streaming
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("autocut server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced shutdown", "err", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
