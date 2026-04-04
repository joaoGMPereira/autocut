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
	"github.com/joaoGMPereira/autocut/server/internal/effects"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/pipeline"
	"github.com/joaoGMPereira/autocut/server/internal/processor"
	"github.com/joaoGMPereira/autocut/server/internal/seed"
	"github.com/joaoGMPereira/autocut/server/internal/thumbnail"
	"github.com/joaoGMPereira/autocut/server/internal/transcript"
)

func main() {
	hostFlag := flag.String("host", "127.0.0.1", "bind host")
	portFlag := flag.Int("port", 4070, "listen port")
	dirFlag := flag.String("dir", "", "data directory")
	migrateOnly := flag.Bool("migrate-only", false, "run DB migrations and exit")
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

	if *migrateOnly {
		logger.Info("migrations complete, exiting")
		return
	}

	// ── Queue auto-recovery (FP-012) ─────────────────────────────────────────
	// Reset uploads interrupted mid-run back to queued so they can be retried.
	if _, recErr := db.ExecContext(context.Background(),
		"UPDATE uploads SET status='queued' WHERE status='running'"); recErr != nil {
		logger.Warn("queue auto-recovery failed (non-fatal)", "err", recErr)
	} else {
		logger.Info("queue auto-recovery: running→queued reset complete")
	}

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

	// ── Seed initial data ────────────────────────────────────────────────────
	// Idempotent: creates channels, channel configs, OAuth secrets, and music
	// files from bundled assets if they don't already exist.
	seedCtx := context.Background()
	if err := seed.New(db, acDir, logger).Run(seedCtx); err != nil {
		logger.Warn("seed failed (non-fatal)", "err", err)
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

	// Processor — pass resolved ffmpeg path from configurator
	ffmpegProc := processor.NewFFmpegProcessor(paths["ffmpeg"])
	shortsGen := processor.NewShortsGenerator(paths["ffmpeg"])
	optimizer := processor.NewVideoOptimizerProcessor(paths["ffmpeg"])

	// Transcript — pass acDir.TransCacheDir and resolved whisper path + best installed model
	whisperTranscriber := transcript.New(transcript.WhisperConfig{
		BinPath:   paths["whisper"],
		ModelPath: toolCfg.BestWhisperModelPath(),
	})
	transcriptCache := transcript.NewCache(acDir.TransCacheDir)

	// AI
	ollamaClient := ai.NewOllamaProvider("http://localhost:11434", 5*time.Minute)
	detector := ai.NewDetector(ollamaClient, ai.DetectorConfig{})

	// Thumbnail — pass resolved ffmpeg and convert (ImageMagick) paths
	thumbnailGen := thumbnail.New(paths["ffmpeg"], paths["convert"])
	// FP-007: background repo for template strategy + background CRUD handler
	bgRepo := database.NewThumbnailBackgroundRepo(db, logger)
	backgroundH := handlers.NewBackgroundHandler(db, logger, dataDir)
	// Legacy pipeline repo — still used by download/processor/transcript/ai/thumbnail/upload handlers
	pipelineRepo := database.NewPipelineRunRepo(db, logger)

	thumbnailH := handlers.NewThumbnailHandler(thumbnailGen, pipelineRepo, bgRepo)

	// Effects — video post-processing effects pipeline
	effectsSvc := effects.NewEffectsService(paths["ffmpeg"])
	effectsH := handlers.NewEffectsHandler(effectsSvc, h)

	// Upload — nil uploader until OAuth is configured
	uploadH := handlers.NewUploadHandler(h, nil, nil, nil, pipelineRepo)

	// Setup handler — wires tool configurator for /api/setup/* routes
	setupH := handlers.NewSetupHandler(h, toolCfg)
	downloadH := handlers.NewDownloadHandler(h, ytDl, twDl, pipelineRepo)
	processorH := handlers.NewProcessorHandler(h, ffmpegProc, shortsGen, optimizer, pipelineRepo)
	transcriptH := handlers.NewTranscriptHandler(h, whisperTranscriber, transcriptCache, pipelineRepo)
	aiH := handlers.NewAIHandler(h, detector, pipelineRepo, paths["ffmpeg"])
	// FP-006: inject thumbnail generator into processor handler for per-short thumbnails.
	processorH.WithThumbnailGenerator(thumbnailGen)

	// Pipeline (rewritten v9 — state-machine executor)
	pipelineNewRepo := pipeline.NewRepo(db)
	chanCfgRepo := database.NewChannelConfigRepo(db, logger)
	pipelineExec := pipeline.NewExecutor(
		pipelineNewRepo,
		h,
		pipeline.NewYouTubeDownloaderAdapter(ytDl),
		pipeline.NewFFmpegCutterAdapter(ffmpegProc),
		pipeline.NewWhisperTranscriberAdapter(whisperTranscriber, transcriptCache, acDir.TransCacheDir),
		pipeline.NewAIAnalyzerAdapter(detector),
		pipeline.NewShortsGeneratorAdapter(shortsGen, ffmpegProc),
		pipeline.NewThumbnailMakerAdapter(thumbnailGen),
		nil,         // Uploader: requires per-channel OAuth tokens
		effectsSvc,  // anti-dup effects
		chanCfgRepo, // channel config reader
	)
	pipelineH := handlers.NewPipelineHandler(pipelineNewRepo, pipelineExec, h, db)

	// Metadata handler
	metadataH := handlers.NewMetadataHandler(ytDl)

	// Stats handler
	statsH := handlers.NewStatsHandler(db)

	// Quota handler — FP-011
	quotaH := handlers.NewQuotaHandler(db)

	// Queue handler — upload queue persistence and scheduling (FP-012/FP-013)
	queueH := handlers.NewQueueHandler(db)

	// YouTube handler — FP-015 (Mass Update) + FP-016 (Comment Sync)
	youtubeH := handlers.NewYouTubeHandler(h, db)

	// OAuth handler — FP-010 (OAuth Multi-Profile Management)
	oauthH := handlers.NewOAuthHandler(db)

	// ── Build router ─────────────────────────────────────────────────────────

	router := api.NewRouter(cfg, db, logger, h,
		downloadH, processorH, transcriptH, aiH, thumbnailH, uploadH, setupH, pipelineH, metadataH, statsH, effectsH,
		backgroundH, queueH, youtubeH, quotaH, oauthH)

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
