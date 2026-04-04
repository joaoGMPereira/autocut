package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	"github.com/spf13/cobra"
)

// baseURL is the HTTP server base URL used by CLI subcommands.
var baseURL string

func main() {
	root := &cobra.Command{
		Use:   "autocut",
		Short: "AutoCut — automated video processing toolkit",
	}

	root.PersistentFlags().StringVar(&baseURL, "server-url", "http://localhost:4070", "AutoCut server base URL")

	root.AddCommand(
		newServerCmd(),
		newDownloadCmd(),
		newCutCmd(),
		newTranscriptCmd(),
		newAnalyzeCmd(),
		newShortsCmd(),
		newThumbnailCmd(),
		newUploadCmd(),
		newOptimizeCmd(),
		newLoginCmd(),
		newMassUpdateCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── server subcommand ─────────────────────────────────────────────────────────

func newServerCmd() *cobra.Command {
	var host string
	var port int
	var dir string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the AutoCut HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(host, port, dir)
		},
	}

	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "bind host")
	cmd.Flags().IntVar(&port, "port", 4070, "listen port")
	cmd.Flags().StringVar(&dir, "dir", "", "data directory (default ~/.autocut)")

	return cmd
}

func runServer(host string, port int, dataDir string) error {
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home dir: %w", err)
		}
		dataDir = fmt.Sprintf("%s/.autocut", home)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	cfg, err := config.Load(dataDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Queue auto-recovery: reset uploads interrupted mid-run back to queued.
	if _, recErr := db.ExecContext(context.Background(),
		"UPDATE uploads SET status='queued' WHERE status='running'"); recErr != nil {
		logger.Warn("queue auto-recovery failed (non-fatal)", "err", recErr)
	} else {
		logger.Info("queue auto-recovery: running→queued reset complete")
	}

	// Init ~/.autocut/ directory structure
	acDir, err := configurator.NewAutoCutDir()
	if err != nil {
		slog.Error("failed to init autocut dir", "err", err)
		return fmt.Errorf("failed to init autocut dir: %w", err)
	}
	if cfg.DataDir != "" {
		acDir = configurator.NewAutoCutDirFromRoot(cfg.DataDir)
	}
	if err := acDir.Ensure(); err != nil {
		slog.Error("failed to create autocut dirs", "err", err)
		return fmt.Errorf("failed to create autocut dirs: %w", err)
	}

	// Seed initial data (idempotent)
	seedCtx := context.Background()
	if err := seed.New(db, acDir, logger).Run(seedCtx); err != nil {
		logger.Warn("seed failed (non-fatal)", "err", err)
	}

	// Init tool configurator
	toolCfg := configurator.New(acDir)
	paths := toolCfg.ResolvedPaths()
	slog.Info("tool paths resolved", "paths", paths)

	// Wire up dependencies
	h := hub.New()
	pipelineRepo := database.NewPipelineRunRepo(db, logger)

	ytDl := downloader.NewYouTubeDownloader(paths["yt-dlp"])
	twDl := downloader.NewTwitchDownloader(paths["TwitchDownloaderCLI"])
	downloadH := handlers.NewDownloadHandler(h, ytDl, twDl, pipelineRepo)

	ffmpegProc := processor.NewFFmpegProcessor(paths["ffmpeg"])
	shortsGen := processor.NewShortsGenerator(paths["ffmpeg"])
	optimizer := processor.NewVideoOptimizerProcessor(paths["ffmpeg"])
	processorH := handlers.NewProcessorHandler(h, ffmpegProc, shortsGen, optimizer, pipelineRepo)

	whisperTranscriber := transcript.New(transcript.WhisperConfig{
		BinPath: paths["whisper"],
	})
	transcriptCache := transcript.NewCache(acDir.TransCacheDir)
	transcriptH := handlers.NewTranscriptHandler(h, whisperTranscriber, transcriptCache, pipelineRepo)

	ollamaClient := ai.NewOllamaProvider("http://localhost:11434", 5*time.Minute)
	detector := ai.NewDetector(ollamaClient, ai.DetectorConfig{})
	aiH := handlers.NewAIHandler(h, detector, pipelineRepo, paths["ffmpeg"])

	thumbnailGen := thumbnail.New(paths["ffmpeg"], paths["convert"])
	bgRepo := database.NewThumbnailBackgroundRepo(db, logger)
	backgroundH := handlers.NewBackgroundHandler(db, logger, dataDir)
	thumbnailH := handlers.NewThumbnailHandler(thumbnailGen, pipelineRepo, bgRepo)
	processorH.WithThumbnailGenerator(thumbnailGen)

	effectsSvc := effects.NewEffectsService(paths["ffmpeg"])
	effectsH := handlers.NewEffectsHandler(effectsSvc, h)

	uploadH := handlers.NewUploadHandler(h, nil, nil, nil, pipelineRepo)

	setupH := handlers.NewSetupHandler(h, toolCfg)

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
	metadataH := handlers.NewMetadataHandler(ytDl)
	statsH := handlers.NewStatsHandler(db)
	queueH := handlers.NewQueueHandler(db)
	youtubeH := handlers.NewYouTubeHandler(h, db)
	quotaH := handlers.NewQuotaHandler(db)
	oauthH := handlers.NewOAuthHandler(db)

	router := api.NewRouter(cfg, db, logger, h,
		downloadH, processorH, transcriptH, aiH, thumbnailH, uploadH, setupH, pipelineH, metadataH, statsH, effectsH,
		backgroundH, queueH, youtubeH, quotaH, oauthH)

	addr := fmt.Sprintf("%s:%d", host, port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
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
		return fmt.Errorf("server forced shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}

// ── download subcommand ───────────────────────────────────────────────────────

func newDownloadCmd() *cobra.Command {
	var quality string
	var chat bool
	var channelID int

	cmd := &cobra.Command{
		Use:   "download <url>",
		Short: "Download a video from a URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"url":     args[0],
				"quality": quality,
				"chat":    chat,
			}
			if channelID != 0 {
				body["channel_id"] = channelID
			}
			resp, err := postJSON(baseURL, "/api/download", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("download job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/download/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().StringVar(&quality, "quality", "1080p", "video quality")
	cmd.Flags().BoolVar(&chat, "chat", false, "include chat download")
	cmd.Flags().IntVar(&channelID, "channel-id", 0, "channel ID")

	return cmd
}

// ── cut subcommand ────────────────────────────────────────────────────────────

func newCutCmd() *cobra.Command {
	var threshold float64
	var minDuration int

	cmd := &cobra.Command{
		Use:   "cut <video-path>",
		Short: "Cut silence from a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"video_path":   args[0],
				"threshold":    threshold,
				"min_duration": minDuration,
			}
			resp, err := postJSON(baseURL, "/api/cut", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("cut job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/cut/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().Float64Var(&threshold, "threshold", 0.6, "silence detection threshold")
	cmd.Flags().IntVar(&minDuration, "min-duration", 30, "minimum segment duration in seconds")

	return cmd
}

// ── transcript subcommand ─────────────────────────────────────────────────────

func newTranscriptCmd() *cobra.Command {
	var language string

	cmd := &cobra.Command{
		Use:   "transcript <video-path>",
		Short: "Generate a transcript for a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"video_path": args[0],
				"language":   language,
			}
			resp, err := postJSON(baseURL, "/api/transcript", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("transcript job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/transcript/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().StringVar(&language, "language", "pt", "transcript language")

	return cmd
}

// ── analyze subcommand ────────────────────────────────────────────────────────

func newAnalyzeCmd() *cobra.Command {
	var strategy string
	var threshold float64

	cmd := &cobra.Command{
		Use:   "analyze <video-path>",
		Short: "Analyze a video for highlights and topics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"video_path": args[0],
				"strategy":   strategy,
				"threshold":  threshold,
			}
			resp, err := postJSON(baseURL, "/api/analyze", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("analyze job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/analyze/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "ai,audio,chat", "analysis strategies (comma-separated)")
	cmd.Flags().Float64Var(&threshold, "threshold", 0.5, "highlight detection threshold")

	return cmd
}

// ── shorts subcommand ─────────────────────────────────────────────────────────

func newShortsCmd() *cobra.Command {
	var blur bool
	var captions bool
	var language string

	cmd := &cobra.Command{
		Use:   "shorts <video-path>",
		Short: "Generate short clips from a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"video_path": args[0],
				"blur":       blur,
				"captions":   captions,
				"language":   language,
			}
			resp, err := postJSON(baseURL, "/api/shorts", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("shorts job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/shorts/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().BoolVar(&blur, "blur", false, "add blur background to vertical shorts")
	cmd.Flags().BoolVar(&captions, "captions", false, "add captions to shorts")
	cmd.Flags().StringVar(&language, "language", "pt", "captions language")

	return cmd
}

// ── thumbnail subcommand ──────────────────────────────────────────────────────

func newThumbnailCmd() *cobra.Command {
	var strategy string
	var channelID int
	var text string

	cmd := &cobra.Command{
		Use:   "thumbnail <video-path>",
		Short: "Generate a thumbnail for a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"video_path": args[0],
				"strategy":   strategy,
				"text":       text,
			}
			if channelID != 0 {
				body["channel_id"] = channelID
			}
			resp, err := postJSON(baseURL, "/api/thumbnail", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("thumbnail job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/thumbnail/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "branded", "thumbnail generation strategy")
	cmd.Flags().IntVar(&channelID, "channel-id", 0, "channel ID")
	cmd.Flags().StringVar(&text, "text", "", "overlay text for the thumbnail")

	return cmd
}

// ── upload subcommand ─────────────────────────────────────────────────────────

func newUploadCmd() *cobra.Command {
	var channelID int
	var schedule string

	cmd := &cobra.Command{
		Use:   "upload <video-path>",
		Short: "Upload a video to YouTube",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"video_path": args[0],
			}
			if channelID != 0 {
				body["channel_id"] = channelID
			}
			if schedule != "" {
				body["schedule"] = schedule
			}
			resp, err := postJSON(baseURL, "/api/upload", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("upload job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/upload/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().IntVar(&channelID, "channel-id", 0, "channel ID")
	cmd.Flags().StringVar(&schedule, "schedule", "", "scheduled publish time (RFC3339, e.g. 2026-01-15T14:00:00Z)")

	return cmd
}

// ── optimize subcommand ───────────────────────────────────────────────────────

func newOptimizeCmd() *cobra.Command {
	var transitions string

	cmd := &cobra.Command{
		Use:   "optimize <video-path>",
		Short: "Optimize and post-process a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"video_path":  args[0],
				"transitions": transitions,
			}
			resp, err := postJSON(baseURL, "/api/optimize", body)
			if err != nil {
				return err
			}
			jobID, ok := extractJobID(resp)
			if !ok {
				return fmt.Errorf("server did not return a job id: %v", resp)
			}
			fmt.Printf("optimize job started: %s\n", jobID)
			return streamSSE(baseURL, fmt.Sprintf("/api/optimize/%s/stream", jobID), printEvent)
		},
	}

	cmd.Flags().StringVar(&transitions, "transitions", "xfade", "transition effect type")

	return cmd
}

// ── login subcommand ──────────────────────────────────────────────────────────

func newLoginCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with YouTube (opens web UI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("OAuth login is managed through the AutoCut web interface at http://localhost:3200")
			if profile != "" {
				fmt.Printf(" (profile: %s)", profile)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "OAuth profile name")

	return cmd
}

// ── mass-update subcommand ────────────────────────────────────────────────────

func newMassUpdateCmd() *cobra.Command {
	var channelID int
	var dryRun bool
	var tagsAdd string
	var tagsRemove string

	cmd := &cobra.Command{
		Use:   "mass-update",
		Short: "Bulk update video metadata on YouTube",
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"dry_run": dryRun,
			}
			if channelID != 0 {
				body["channel_id"] = channelID
			}
			if tagsAdd != "" {
				body["tags_add"] = strings.Split(tagsAdd, ",")
			}
			if tagsRemove != "" {
				body["tags_remove"] = strings.Split(tagsRemove, ",")
			}
			resp, err := postJSON(baseURL, "/api/videos/mass-update", body)
			if err != nil {
				return err
			}
			out, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}

	cmd.Flags().IntVar(&channelID, "channel-id", 0, "channel ID")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")
	cmd.Flags().StringVar(&tagsAdd, "tags-add", "", "comma-separated tags to add")
	cmd.Flags().StringVar(&tagsRemove, "tags-remove", "", "comma-separated tags to remove")

	return cmd
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// postJSON encodes body as JSON, POSTs to baseURL+path, and decodes the JSON response.
func postJSON(base, path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	resp, err := http.Post(base+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg, _ := result["error"].(string)
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, msg)
	}

	return result, nil
}

// streamSSE connects to a Server-Sent Events endpoint and calls onEvent for
// each complete event. It returns when it receives a "done" or "error" event
// type, or when the connection closes.
func streamSSE(base, path string, onEvent func(event, data string)) error {
	resp, err := http.Get(base + path)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stream error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	var eventType, eventData string

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			eventData = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		case line == "":
			// Blank line = dispatch the event
			if eventType != "" || eventData != "" {
				onEvent(eventType, eventData)
				if eventType == "done" || eventType == "error" {
					return nil
				}
			}
			eventType = ""
			eventData = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}
	return nil
}

// printEvent prints an SSE event to stdout in the format: [event] data: message
func printEvent(event, data string) {
	if event == "" {
		event = "message"
	}
	fmt.Printf("[%s] %s\n", event, data)
	slog.Debug("sse event", "event", event, "data", data)
}

// extractJobID attempts to extract a job ID from a server response map.
// It checks "job_id", "id", and "pipeline_run_id" keys.
func extractJobID(resp map[string]interface{}) (string, bool) {
	for _, key := range []string{"job_id", "id", "pipeline_run_id"} {
		if v, ok := resp[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}
