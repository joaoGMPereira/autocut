package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// Service orchestrates pipeline run state transitions and background download.
type Service struct {
	repo        *database.PipelineRunRepo
	historyRepo *database.URLHistoryRepo
	hub         *hub.SSEHub
	ytDl        *downloader.YouTubeDownloader
	twDl        *downloader.TwitchDownloader
	dataDir     string
	log         *slog.Logger
	cancelMap   sync.Map // map[int64]context.CancelFunc — one entry per active run
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
	h *hub.SSEHub,
	ytDl *downloader.YouTubeDownloader,
	twDl *downloader.TwitchDownloader,
	dataDir string,
) *Service {
	return &Service{
		repo:        repo,
		historyRepo: historyRepo,
		hub:         h,
		ytDl:        ytDl,
		twDl:        twDl,
		dataDir:     dataDir,
		log:         slog.With("component", "pipeline.service"),
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

// Advance resolves the current gate state. Polymorphic — behavior depends on run's current state.
func (s *Service) Advance(ctx context.Context, id int64, req AdvanceRequest) (string, error) {
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("run %d not found", id)
		}
		return "", fmt.Errorf("get run %d: %w", id, err)
	}

	// Terminal states — idempotent.
	if run.State == StateDone || run.State == StateError || run.State == StateCancelled {
		return run.State, nil
	}

	jobKey := fmt.Sprintf("%d", id)

	switch run.State {
	case StateWaitingURL:
		if req.URL == "" {
			return "", fmt.Errorf("url is required")
		}
		normalized, err := normalizeURL(req.URL)
		if err != nil {
			return "", err
		}
		if err := s.repo.StartDownload(ctx, id, normalized); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", fmt.Errorf("run %d not found or not in WAITING_URL state", id)
			}
			return "", fmt.Errorf("advance run %d: %w", id, err)
		}
		s.log.Info("pipeline run advancing", "run_id", id, "url", normalized)
		runCtx, cancel := context.WithCancel(context.Background())
		s.cancelMap.Store(id, cancel)
		go s.runDownload(runCtx, id, normalized)
		return StateWaitingURL, nil

	case StateWaitingMode:
		// Validate mode field from submitted config
		var configMap map[string]any
		if len(req.ModeConfigJSON) > 0 {
			if err := json.Unmarshal(req.ModeConfigJSON, &configMap); err != nil {
				return "", fmt.Errorf("invalid mode_config JSON: %w", err)
			}
		}
		modeVal, _ := configMap["mode"].(string)
		if modeVal != "ai" && modeVal != "longform" {
			return "", fmt.Errorf("invalid mode %q — must be 'ai' or 'longform'", modeVal)
		}

		// Persist mode config before advancing state
		if err := s.repo.StoreModeConfig(ctx, id, modeVal, req.ModeConfigJSON); err != nil {
			// Non-fatal for empty config (first-time run with no config)
			s.log.Warn("failed to store mode config", "run_id", id, "err", err)
		}

		if err := s.repo.AdvanceState(ctx, id, StateWaitingMode, StateExecuting); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
					return curr.State, nil
				}
			}
			return "", fmt.Errorf("advance run %d: %w", id, err)
		}
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "state_changed",
			Data: stateChangedPayload{RunID: id, State: StateExecuting},
		})
		go s.runProcessingStub(id)
		return StateExecuting, nil

	case StateWaitingReviewHighlights:
		if err := s.repo.AdvanceState(ctx, id, StateWaitingReviewHighlights, StateGeneratingClips); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
					return curr.State, nil
				}
			}
			return "", fmt.Errorf("advance run %d: %w", id, err)
		}
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "state_changed",
			Data: stateChangedPayload{RunID: id, State: StateGeneratingClips},
		})
		go s.runClipGenStub(id)
		return StateGeneratingClips, nil

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
					return curr.State, nil
				}
			}
			return "", fmt.Errorf("advance run %d: %w", id, err)
		}
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "state_changed",
			Data: stateChangedPayload{RunID: id, State: StateUploading},
		})
		go s.runUploadStub(id)
		return StateUploading, nil

	default:
		return "", fmt.Errorf("no advance action for state %s", run.State)
	}
}

// advanceSimple atomically moves a run from fromState to toState (no background goroutine).
func (s *Service) advanceSimple(ctx context.Context, id int64, fromState, toState string) (string, error) {
	jobKey := fmt.Sprintf("%d", id)
	if err := s.repo.AdvanceState(ctx, id, fromState, toState); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
				return curr.State, nil
			}
		}
		return "", fmt.Errorf("advance run: %w", err)
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: toState},
	})
	return toState, nil
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

	// Transition to WAITING_MODE.
	if err := s.repo.Finish(ctx, id, StateWaitingMode, ""); err != nil {
		s.log.Error("finish run failed", "run_id", id, "err", err)
	}

	// Publish 100% progress then state change.
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "phase_progress",
		Data: phaseProgressPayload{RunID: id, Phase: PhaseDownload, PercentDone: 100},
	})
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: StateWaitingMode},
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

// runClipGenStub advances GENERATING_CLIPS → WAITING_THUMBNAIL_CONFIG after a 100ms delay.
func (s *Service) runClipGenStub(id int64) {
	time.Sleep(100 * time.Millisecond)
	jobKey := fmt.Sprintf("%d", id)
	ctx := context.Background()
	if err := s.repo.AdvanceState(ctx, id, StateGeneratingClips, StateWaitingThumbnailConfig); err != nil {
		s.log.Error("clip gen stub advance failed", "run_id", id, "err", err)
		return
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: StateWaitingThumbnailConfig},
	})
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
