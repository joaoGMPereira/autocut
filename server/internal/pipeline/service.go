package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/processor"
)

// Service orchestrates pipeline run state transitions and background download.
type Service struct {
	repo          *database.PipelineRunRepo
	historyRepo   *database.URLHistoryRepo
	highlightRepo *database.PipelineHighlightRepo
	clipRepo      *database.PipelineClipRepo
	channelCfgRepo *database.ChannelConfigRepo
	hub           *hub.SSEHub
	ytDl          *downloader.YouTubeDownloader
	twDl          *downloader.TwitchDownloader
	dataDir       string
	log           *slog.Logger
	cancelMap     sync.Map // map[int64]context.CancelFunc — one entry per active run
}

// videoInfoPayload is the SSE payload for the video_info event.
type videoInfoPayload struct {
	RunID        int64  `json:"run_id"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	DurationSec  int    `json:"duration_sec"`
	ChannelName  string `json:"channel_name,omitempty"`
}

// phaseProgressPayload is the SSE payload for phase_progress events.
type phaseProgressPayload struct {
	RunID       int64   `json:"run_id"`
	Phase       string  `json:"phase"`
	PercentDone float64 `json:"percent_done"`
	SpeedKbs    *int    `json:"speed_kbs,omitempty"`
	EtaSec      *int    `json:"eta_sec,omitempty"`
}

// stateChangedPayload is the SSE payload for state_changed events.
type stateChangedPayload struct {
	RunID int64  `json:"run_id"`
	State string `json:"state"`
}

// downloadCompletePayload is the SSE payload for the download_complete event.
type downloadCompletePayload struct {
	RunID     int64  `json:"run_id"`
	VideoPath string `json:"video_path"`
}

// AdvanceRequest holds all possible gate-advance payloads.
// Fields are interpreted based on the run's current state.
type AdvanceRequest struct {
	URL            string `json:"url"`
	Mode           string `json:"mode"`
	ModeConfigJSON json.RawMessage `json:"mode_config"`
}

// NewService constructs a PipelineService.
func NewService(
	repo *database.PipelineRunRepo,
	historyRepo *database.URLHistoryRepo,
	highlightRepo *database.PipelineHighlightRepo,
	clipRepo *database.PipelineClipRepo,
	channelCfgRepo *database.ChannelConfigRepo,
	h *hub.SSEHub,
	ytDl *downloader.YouTubeDownloader,
	twDl *downloader.TwitchDownloader,
	dataDir string,
) *Service {
	return &Service{
		repo:           repo,
		historyRepo:    historyRepo,
		highlightRepo:  highlightRepo,
		clipRepo:       clipRepo,
		channelCfgRepo: channelCfgRepo,
		hub:            h,
		ytDl:           ytDl,
		twDl:           twDl,
		dataDir:        dataDir,
		log:            slog.With("component", "pipeline.service"),
	}
}

// Create inserts a new pipeline run in WAITING_URL state and returns it.
func (s *Service) Create(ctx context.Context) (*database.PipelineRun, error) {
	run := &database.PipelineRun{State: StateWaitingURL}
	id, err := s.repo.Create(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("create pipeline run: %w", err)
	}
	run.ID = id
	s.log.Info("pipeline run created", "run_id", id)
	return run, nil
}

// HasPriorDoneRun checks if a prior completed run exists for the given URL.
// Returns (true, runID) if found, (false, 0) otherwise.
func (s *Service) HasPriorDoneRun(ctx context.Context, url string) (bool, int64, error) {
	return s.repo.HasPriorDoneRun(ctx, url)
}

// GetByID returns the pipeline run with the given ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*database.PipelineRun, error) {
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get pipeline run %d: %w", id, err)
	}
	return run, nil
}

// AdvanceResult holds the outcome of an Advance call.
type AdvanceResult struct {
	State       string `json:"state"`
	VideoReused bool   `json:"video_reused"`
}

// Advance resolves the current gate state. Polymorphic — behavior depends on run's current state.
func (s *Service) Advance(ctx context.Context, id int64, req AdvanceRequest) (AdvanceResult, error) {
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AdvanceResult{}, fmt.Errorf("run %d not found", id)
		}
		return AdvanceResult{}, fmt.Errorf("get run %d: %w", id, err)
	}

	// Terminal states — idempotent.
	if run.State == StateDone || run.State == StateError || run.State == StateCancelled {
		return AdvanceResult{State: run.State}, nil
	}

	jobKey := fmt.Sprintf("%d", id)

	switch run.State {
	case StateWaitingURL:
		if req.URL == "" {
			return AdvanceResult{}, fmt.Errorf("url is required")
		}
		normalized, err := normalizeURL(req.URL)
		if err != nil {
			return AdvanceResult{}, err
		}

		// Check if a prior run already downloaded this URL and the file still exists.
		existingPath, existingTitle, existingDuration, found, findErr := s.repo.FindExistingVideo(ctx, normalized)
		if findErr != nil {
			s.log.Warn("find existing video failed, proceeding with fresh download", "run_id", id, "err", findErr)
		}

		if found && existingPath != "" {
			if _, statErr := os.Stat(existingPath); statErr == nil {
				// Reuse: copy file to new run's pipeline dir.
				destDir := fmt.Sprintf("%s/downloads/%d", s.dataDir, id)
				if mkErr := os.MkdirAll(destDir, 0o755); mkErr != nil {
					return AdvanceResult{}, fmt.Errorf("create download dir: %w", mkErr)
				}
				destPath := filepath.Join(destDir, filepath.Base(existingPath))
				if cpErr := copyFile(existingPath, destPath); cpErr != nil {
					s.log.Warn("video copy failed, falling back to fresh download", "run_id", id, "err", cpErr)
				} else {
					// Transition WAITING_URL → WAITING_MODE with no active_phase (download done).
					if err := s.repo.StartDownloadAndAdvance(ctx, id, normalized); err != nil {
						if errors.Is(err, sql.ErrNoRows) {
							return AdvanceResult{}, fmt.Errorf("run %d not found or not in WAITING_URL state", id)
						}
						return AdvanceResult{}, fmt.Errorf("advance run %d: %w", id, err)
					}
					// Set download result (video_path, title, duration).
					if err := s.repo.SetDownloadResult(ctx, id, destPath, existingTitle, existingDuration); err != nil {
						s.log.Error("set download result failed after reuse", "run_id", id, "err", err)
					}
					// Clear active_phase since download is already done.
					if err := s.repo.SetActivePhase(ctx, id, ""); err != nil {
						s.log.Error("clear active_phase failed after reuse", "run_id", id, "err", err)
					}

					s.log.Info("pipeline run reused video", "run_id", id, "url", normalized, "source", existingPath)
					s.hub.Publish(jobKey, hub.SSEEvent{
						Type: "state_changed",
						Data: stateChangedPayload{RunID: id, State: StateWaitingMode},
					})
					return AdvanceResult{State: StateWaitingMode, VideoReused: true}, nil
				}
			}
		}

		// No reuse — start fresh download. Transition to WAITING_MODE immediately with active_phase="download".
		if err := s.repo.StartDownloadAndAdvance(ctx, id, normalized); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return AdvanceResult{}, fmt.Errorf("run %d not found or not in WAITING_URL state", id)
			}
			return AdvanceResult{}, fmt.Errorf("advance run %d: %w", id, err)
		}
		s.log.Info("pipeline run advancing", "run_id", id, "url", normalized)
		runCtx, cancel := context.WithCancel(context.Background())
		s.cancelMap.Store(id, cancel)
		go s.runDownload(runCtx, id, normalized)
		return AdvanceResult{State: StateWaitingMode, VideoReused: false}, nil

	case StateWaitingMode:
		// Validate mode field from submitted config
		var configMap map[string]any
		if len(req.ModeConfigJSON) > 0 {
			if err := json.Unmarshal(req.ModeConfigJSON, &configMap); err != nil {
				return AdvanceResult{}, fmt.Errorf("invalid mode_config JSON: %w", err)
			}
		}
		modeVal, _ := configMap["mode"].(string)
		if modeVal != "ai" && modeVal != "longform" {
			return AdvanceResult{}, fmt.Errorf("invalid mode %q — must be 'ai' or 'longform'", modeVal)
		}

		// Persist mode config before advancing state
		if err := s.repo.StoreModeConfig(ctx, id, modeVal, req.ModeConfigJSON); err != nil {
			// Non-fatal for empty config (first-time run with no config)
			s.log.Warn("failed to store mode config", "run_id", id, "err", err)
		}

		if err := s.repo.AdvanceState(ctx, id, StateWaitingMode, StateExecuting); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
					return AdvanceResult{State: curr.State}, nil
				}
			}
			return AdvanceResult{}, fmt.Errorf("advance run %d: %w", id, err)
		}
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "state_changed",
			Data: stateChangedPayload{RunID: id, State: StateExecuting},
		})
		go s.runProcessingStub(id)
		return AdvanceResult{State: StateExecuting}, nil

	case StateWaitingReviewHighlights:
		if err := s.repo.AdvanceState(ctx, id, StateWaitingReviewHighlights, StateGeneratingClips); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
					return AdvanceResult{State: curr.State}, nil
				}
			}
			return AdvanceResult{}, fmt.Errorf("advance run %d: %w", id, err)
		}
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "state_changed",
			Data: stateChangedPayload{RunID: id, State: StateGeneratingClips},
		})
		go s.runClipGeneration(id)
		return AdvanceResult{State: StateGeneratingClips}, nil

	case StateWaitingThumbnailConfig:
		return s.advanceSimple(ctx, id, StateWaitingThumbnailConfig, StateWaitingReviewMetadata)

	case StateWaitingReviewMetadata:
		return s.advanceSimple(ctx, id, StateWaitingReviewMetadata, StateWaitingReviewClips)

	case StateWaitingReviewClips:
		return s.advanceSimple(ctx, id, StateWaitingReviewClips, StateWaitingUploadConfirm)

	case StateWaitingUploadConfirm:
		if err := s.repo.AdvanceState(ctx, id, StateWaitingUploadConfirm, StateUploading); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
					return AdvanceResult{State: curr.State}, nil
				}
			}
			return AdvanceResult{}, fmt.Errorf("advance run %d: %w", id, err)
		}
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "state_changed",
			Data: stateChangedPayload{RunID: id, State: StateUploading},
		})
		go s.runUploadStub(id)
		return AdvanceResult{State: StateUploading}, nil

	default:
		return AdvanceResult{}, fmt.Errorf("no advance action for state %s", run.State)
	}
}

// advanceSimple atomically moves a run from fromState to toState (no background goroutine).
func (s *Service) advanceSimple(ctx context.Context, id int64, fromState, toState string) (AdvanceResult, error) {
	jobKey := fmt.Sprintf("%d", id)
	if err := s.repo.AdvanceState(ctx, id, fromState, toState); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
				return AdvanceResult{State: curr.State}, nil
			}
		}
		return AdvanceResult{}, fmt.Errorf("advance run: %w", err)
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: toState},
	})
	return AdvanceResult{State: toState}, nil
}

// Redownload clears the current video_path and starts a fresh download.
// Only valid when the run is in WAITING_MODE state.
func (s *Service) Redownload(ctx context.Context, id int64) error {
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("run %d not found", id)
		}
		return fmt.Errorf("get run %d: %w", id, err)
	}
	if run.State != StateWaitingMode {
		return fmt.Errorf("redownload only valid in WAITING_MODE state, current state: %s", run.State)
	}
	if run.URL == "" {
		return fmt.Errorf("run %d has no URL to redownload", id)
	}

	if err := s.repo.SetVideoPath(ctx, id, ""); err != nil {
		return fmt.Errorf("clear video_path: %w", err)
	}
	if err := s.repo.SetActivePhase(ctx, id, PhaseDownload); err != nil {
		return fmt.Errorf("set active_phase: %w", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.cancelMap.Store(id, cancel)
	go s.runDownload(runCtx, id, run.URL)

	s.log.Info("pipeline run redownload started", "run_id", id, "url", run.URL)
	return nil
}

// Cancel transitions a run to CANCELLED and terminates the background download.
func (s *Service) Cancel(ctx context.Context, id int64) (string, error) {
	if err := s.repo.Cancel(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("run %d not found or already in terminal state", id)
		}
		return "", fmt.Errorf("cancel run %d: %w", id, err)
	}
	// Signal the goroutine to stop.
	if fn, loaded := s.cancelMap.LoadAndDelete(id); loaded {
		fn.(context.CancelFunc)()
	}
	jobKey := fmt.Sprintf("%d", id)
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "cancelled",
		Data: stateChangedPayload{RunID: id, State: StateCancelled},
	})
	s.log.Info("pipeline run cancelled", "run_id", id)
	return StateCancelled, nil
}

// normalizeURL returns the canonical form of a YouTube or Twitch URL.
// Returns an error for invalid or unsupported URLs.
func normalizeURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid url: %q", rawURL)
	}
	host := strings.ToLower(u.Host)

	if host == "youtu.be" {
		id := strings.TrimPrefix(u.Path, "/")
		if id == "" {
			return "", fmt.Errorf("invalid youtu.be url: no video id")
		}
		return "https://www.youtube.com/watch?v=" + id, nil
	}

	if strings.HasSuffix(host, "youtube.com") {
		if strings.HasPrefix(u.Path, "/watch") {
			v := u.Query().Get("v")
			if v == "" {
				return "", fmt.Errorf("invalid youtube watch url: no v param")
			}
			return "https://www.youtube.com/watch?v=" + v, nil
		}
		if strings.HasPrefix(u.Path, "/shorts/") {
			id := strings.TrimPrefix(u.Path, "/shorts/")
			if id == "" {
				return "", fmt.Errorf("invalid youtube shorts url: no id")
			}
			return "https://www.youtube.com/watch?v=" + id, nil
		}
	}

	if strings.HasSuffix(host, "twitch.tv") {
		parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
		channel := strings.ToLower(parts[0])
		if channel == "" {
			return "", fmt.Errorf("invalid twitch url: no channel")
		}
		return "https://www.twitch.tv/" + channel, nil
	}

	return "", fmt.Errorf("unsupported platform: %q — only YouTube and Twitch URLs are supported", u.Host)
}

// runDownload is the background goroutine that executes the download phase.
func (s *Service) runDownload(ctx context.Context, id int64, videoURL string) {
	jobKey := fmt.Sprintf("%d", id)
	defer s.cancelMap.Delete(id)

	publishError := func(msg string) {
		s.log.Error("pipeline download failed", "run_id", id, "err", msg)
		_ = s.repo.Finish(ctx, id, StateError, msg)
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "error",
			Data: map[string]interface{}{"run_id": id, "state": StateError, "message": msg},
		})
	}

	// Publish download start (0%).
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "phase_progress",
		Data: phaseProgressPayload{RunID: id, Phase: PhaseDownload, PercentDone: 0},
	})

	// Prepare output directory.
	outDir := fmt.Sprintf("%s/downloads/%d", s.dataDir, id)

	isTwitch := strings.Contains(videoURL, "twitch.tv")

	// Extract metadata first.
	var metaInfo downloader.VideoInfo
	var metaErr error
	if isTwitch {
		metaInfo, metaErr = s.twDl.ExtractMetadata(videoURL)
	} else {
		metaInfo, metaErr = s.ytDl.ExtractMetadata(videoURL)
	}
	if metaErr != nil {
		if ctx.Err() != nil {
			return // cancelled before metadata finished
		}
		publishError(fmt.Sprintf("extract metadata: %s", metaErr))
		return
	}

	// Publish video_info as soon as metadata is available.
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "video_info",
		Data: videoInfoPayload{
			RunID:        id,
			Title:        metaInfo.Title,
			ThumbnailURL: metaInfo.ThumbnailURL,
			DurationSec:  int(metaInfo.Duration.Seconds()),
			ChannelName:  metaInfo.ChannelName,
		},
	})

	// onProgress: stream phase_progress events for each yt-dlp line.
	onProgress := func(pct float64, speedKbs, etaSec *int) {
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "phase_progress",
			Data: phaseProgressPayload{
				RunID:       id,
				Phase:       PhaseDownload,
				PercentDone: pct,
				SpeedKbs:    speedKbs,
				EtaSec:      etaSec,
			},
		})
	}

	// Download video with context (supports cancellation + live progress).
	var dlInfo downloader.VideoInfo
	var dlErr error
	if isTwitch {
		// Twitch downloader does not yet have streaming support; fall back.
		_, dlErr = s.twDl.DownloadWithOptions(videoURL, outDir, "")
		if dlErr == nil {
			dlInfo = metaInfo
			dlInfo.FilePath = fmt.Sprintf("%s/%s.mp4", outDir, metaInfo.VideoID)
		}
	} else {
		dlInfo, dlErr = s.ytDl.DownloadWithContext(ctx, videoURL, outDir, "", onProgress)
	}

	if dlErr != nil {
		if ctx.Err() != nil {
			return // context cancelled — Cancel() already wrote CANCELLED state
		}
		publishError(fmt.Sprintf("download: %s", dlErr))
		return
	}

	// Guard: video_path must be non-empty.
	if dlInfo.FilePath == "" {
		publishError("download completed but video_path is empty — cannot proceed")
		return
	}

	// Persist download result atomically.
	durationSec := int64(dlInfo.Duration.Seconds())
	if durationSec <= 0 {
		durationSec = int64(metaInfo.Duration.Seconds())
	}
	title := dlInfo.Title
	if title == "" {
		title = metaInfo.Title
	}
	if err := s.repo.SetDownloadResult(ctx, id, dlInfo.FilePath, title, durationSec); err != nil {
		publishError(fmt.Sprintf("persist download result: %s", err))
		return
	}

	// Persist URL in history after successful download.
	if s.historyRepo != nil {
		if err := s.historyRepo.Upsert(ctx, database.URLHistoryEntry{
			URL:        videoURL,
			VideoTitle: func() *string { if title == "" { return nil }; return &title }(),
			LastUsedAt: time.Now().UnixMilli(),
		}); err != nil {
			s.log.Warn("upsert url_history failed", "run_id", id, "err", err)
		}
	}

	// Clear active_phase — download is done. State is already WAITING_MODE (set during Advance).
	if err := s.repo.SetActivePhase(ctx, id, ""); err != nil {
		s.log.Error("clear active_phase failed", "run_id", id, "err", err)
	}

	// Publish 100% progress then download_complete event.
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "phase_progress",
		Data: phaseProgressPayload{RunID: id, Phase: PhaseDownload, PercentDone: 100},
	})
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "download_complete",
		Data: downloadCompletePayload{RunID: id, VideoPath: dlInfo.FilePath},
	})

	s.log.Info("pipeline download complete", "run_id", id, "duration", metaInfo.Duration.Round(time.Second))
}

// runProcessingStub advances EXECUTING → WAITING_REVIEW_HIGHLIGHTS after a 100ms delay.
func (s *Service) runProcessingStub(id int64) {
	time.Sleep(100 * time.Millisecond)
	jobKey := fmt.Sprintf("%d", id)
	ctx := context.Background()
	if err := s.repo.AdvanceState(ctx, id, StateExecuting, StateWaitingReviewHighlights); err != nil {
		s.log.Error("processing stub advance failed", "run_id", id, "err", err)
		return
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: StateWaitingReviewHighlights},
	})
}

// runClipGeneration generates clips for all selected highlights with the full effect pipeline.
func (s *Service) runClipGeneration(id int64) {
	jobKey := fmt.Sprintf("%d", id)
	ctx := context.Background()
	log := s.log.With("run_id", id, "phase", "clip_gen")

	publishError := func(msg string) {
		log.Error("clip generation failed", "err", msg)
		_ = s.repo.Finish(ctx, id, StateError, msg)
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "error",
			Data: map[string]interface{}{"run_id": id, "state": StateError, "message": msg},
		})
	}

	// Load run for video_path, duration, channel, mode config
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		publishError(fmt.Sprintf("load run: %s", err))
		return
	}
	if run.VideoPath == "" {
		publishError("no video_path — download may have failed")
		return
	}

	// Load channel config for effect layers
	var channelCfg database.ChannelConfig
	if run.ChannelID.Valid && s.channelCfgRepo != nil {
		cc, err := s.channelCfgRepo.GetByChannelID(ctx, run.ChannelID.Int64)
		if err != nil {
			log.Warn("channel config lookup failed, using defaults", "channel_id", run.ChannelID.Int64, "err", err)
		} else {
			channelCfg = *cc
		}
	}

	// Fetch selected highlights
	highlights, err := s.highlightRepo.ListSelectedByRun(ctx, id)
	if err != nil {
		publishError(fmt.Sprintf("load highlights: %s", err))
		return
	}
	if len(highlights) == 0 {
		log.Warn("no selected highlights — advancing without clips")
		s.advanceClipGenDone(id)
		return
	}

	totalClips := len(highlights)
	log.Info("starting clip generation", "total_clips", totalClips)

	for i, hl := range highlights {
		// Use adjusted boundaries if set, otherwise original
		startSec := hl.AdjStartSec
		endSec := hl.AdjEndSec
		if startSec == 0 && endSec == 0 {
			startSec = hl.StartSec
			endSec = hl.EndSec
		}

		// Create clip row
		clip := &database.PipelineClip{
			RunID:       id,
			HighlightID: sql.NullInt64{Int64: hl.ID, Valid: true},
			StartSec:    startSec,
			EndSec:      endSec,
			DurationSec: endSec - startSec,
			IsSelected:  true,
			Title:       hl.Text,
		}
		clipID, err := s.clipRepo.Create(ctx, clip)
		if err != nil {
			log.Error("create clip row failed", "highlight_id", hl.ID, "err", err)
			continue
		}

		// Publish per-clip progress
		clipPct := float64(i) / float64(totalClips) * 100
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "phase_progress",
			Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: clipPct},
		})

		// Generate clip with all effect layers
		clipReq := processor.ClipRequest{
			RunID:      id,
			ClipID:     clipID,
			VideoPath:  run.VideoPath,
			StartSec:   startSec,
			EndSec:     endSec,
			ChannelCfg: channelCfg,
			ModeCfg:    json.RawMessage(run.ModeConfigJSON),
			DataDir:    s.dataDir,
			OnProgress: func(pct float64) {
				// Scale per-clip progress within the overall range
				overallPct := (float64(i) + pct/100) / float64(totalClips) * 100
				s.hub.Publish(jobKey, hub.SSEEvent{
					Type: "phase_progress",
					Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: overallPct},
				})
			},
		}

		outPath, err := processor.GenerateClip(ctx, clipReq)
		if err != nil {
			log.Error("clip generation failed", "clip_id", clipID, "highlight_id", hl.ID, "err", err)
			_ = s.clipRepo.UpdateStatus(ctx, clipID, "error")
			continue // continue with next clip
		}

		// Update clip with file path
		_ = s.clipRepo.UpdateFilePath(ctx, clipID, outPath)
		_ = s.clipRepo.UpdateStatus(ctx, clipID, "ready")
		log.Info("clip generated", "clip_id", clipID, "path", outPath, "progress", fmt.Sprintf("%d/%d", i+1, totalClips))
	}

	// Publish 100% progress
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "phase_progress",
		Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: 100},
	})

	s.advanceClipGenDone(id)
}

// advanceClipGenDone transitions from GENERATING_CLIPS to WAITING_THUMBNAIL_CONFIG.
func (s *Service) advanceClipGenDone(id int64) {
	jobKey := fmt.Sprintf("%d", id)
	ctx := context.Background()
	if err := s.repo.AdvanceState(ctx, id, StateGeneratingClips, StateWaitingThumbnailConfig); err != nil {
		s.log.Error("clip gen advance failed", "run_id", id, "err", err)
		return
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: StateWaitingThumbnailConfig},
	})
}

// copyFile copies src to dst using io.Copy.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return out.Close()
}

// runUploadStub advances UPLOADING → DONE after a 100ms delay.
func (s *Service) runUploadStub(id int64) {
	time.Sleep(100 * time.Millisecond)
	jobKey := fmt.Sprintf("%d", id)
	ctx := context.Background()
	if err := s.repo.AdvanceState(ctx, id, StateUploading, StateDone); err != nil {
		s.log.Error("upload stub advance failed", "run_id", id, "err", err)
		return
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: StateDone},
	})
}
