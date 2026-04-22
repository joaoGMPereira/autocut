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
	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

// ErrWrongState is returned when an operation cannot proceed because the run is not in the expected state.
var ErrWrongState = errors.New("wrong state")

// metaGeneratorIface is the subset of ai.MetadataGenerator used by the pipeline service.
type metaGeneratorIface interface {
	Available() bool
	PrefillBatch(ctx context.Context, runID int64)
}

// Service orchestrates pipeline run state transitions and background download.
type Service struct {
	repo          *database.PipelineRunRepo
	historyRepo   *database.URLHistoryRepo
	highlightRepo *database.PipelineHighlightRepo
	clipRepo      *database.PipelineClipRepo
	channelCfgRepo *database.ChannelConfigRepo
	settingRepo    *database.AppSettingRepo
	uploadRepo     *database.UploadRepo      // NEW
	queueStorage   *uploader.QueueStorage    // NEW
	hub           *hub.SSEHub
	ytDl          *downloader.YouTubeDownloader
	twDl          *downloader.TwitchDownloader
	dataDir       string
	log           *slog.Logger
	cancelMap     sync.Map // map[int64]context.CancelFunc — one entry per active run
	metaGenerator metaGeneratorIface  // NEW — nil if Claude unavailable
}

// SetMetadataGenerator sets the metadata generator. Called from main.go after construction
// to avoid circular initialization.
func (s *Service) SetMetadataGenerator(g metaGeneratorIface) {
	s.metaGenerator = g
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
	ClipID      *int64  `json:"clip_id,omitempty"`
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
	URL            string          `json:"url"`
	ChannelID      *int64          `json:"channel_id"`
	Mode           string          `json:"mode"`
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
	settingRepo *database.AppSettingRepo,
	uploadRepo *database.UploadRepo,
	queueStorage *uploader.QueueStorage,
) *Service {
	return &Service{
		repo:           repo,
		historyRepo:    historyRepo,
		highlightRepo:  highlightRepo,
		clipRepo:       clipRepo,
		channelCfgRepo: channelCfgRepo,
		settingRepo:    settingRepo,
		uploadRepo:     uploadRepo,
		queueStorage:   queueStorage,
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

// HasPriorRunWithClips checks if any prior run for the given URL has ready clips.
func (s *Service) HasPriorRunWithClips(ctx context.Context, url string) (bool, int64, error) {
	return s.repo.HasPriorRunWithClips(ctx, url)
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

// ListClipsByRun returns all clips for a pipeline run.
func (s *Service) ListClipsByRun(ctx context.Context, runID int64) ([]database.PipelineClip, error) {
	return s.clipRepo.ListByRun(ctx, runID)
}

// AdvanceResult holds the outcome of an Advance call.
type AdvanceResult struct {
	State       string `json:"state"`
	VideoReused bool   `json:"video_reused"`
}

// ReviewClipsPayload is the gate request for WAITING_REVIEW_CLIPS.
type ReviewClipsPayload struct {
	SelectedIDs []int64        `json:"selected_ids"`
	ClipEdits   []ClipTextEdit `json:"clip_edits"`
}

// ClipTextEdit carries inline title/thumbnail_text edits from the review screen.
type ClipTextEdit struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	ThumbnailText string `json:"thumbnail_text"`
}

// ReviewClips persists clip selection and text edits, then advances to WAITING_UPLOAD_CONFIRM.
func (s *Service) ReviewClips(ctx context.Context, runID int64, payload ReviewClipsPayload) (string, error) {
	run, err := s.repo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("run %d not found: %w", runID, sql.ErrNoRows)
		}
		return "", fmt.Errorf("get run %d: %w", runID, err)
	}
	if run.State != StateWaitingReviewClips {
		return "", fmt.Errorf("run not in %s state (current: %s): %w", StateWaitingReviewClips, run.State, ErrWrongState)
	}

	if len(payload.SelectedIDs) == 0 {
		return "", fmt.Errorf("at least one clip must be selected")
	}

	// Persist selections
	if err := s.clipRepo.UpdateIsSelectedBatch(ctx, runID, payload.SelectedIDs); err != nil {
		return "", fmt.Errorf("update is_selected: %w", err)
	}

	// Save any inline text edits (non-fatal per clip)
	for _, edit := range payload.ClipEdits {
		if err := s.clipRepo.UpdateTitleAndText(ctx, runID, edit.ID, edit.Title, edit.ThumbnailText); err != nil {
			s.log.Warn("failed to update clip text", "clip_id", edit.ID, "err", err)
		}
	}

	// Advance state
	if err := s.repo.AdvanceState(ctx, runID, StateWaitingReviewClips, StateWaitingUploadConfirm); err != nil {
		return "", fmt.Errorf("advance state: %w", err)
	}

	// Notify SSE subscribers
	s.hub.Publish(fmt.Sprintf("%d", runID), hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: runID, State: StateWaitingUploadConfirm},
	})

	return StateWaitingUploadConfirm, nil
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
		existingPath, existingTitle, existingDuration, existingThumbURL, found, findErr := s.repo.FindExistingVideo(ctx, normalized)
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
					if req.ChannelID != nil {
						if err := s.repo.SetChannelID(ctx, id, *req.ChannelID); err != nil {
							s.log.Warn("failed to save channel_id on run", "run_id", id, "err", err)
						}
					}
					// Set download result (video_path, title, duration, thumbnail_url).
					if err := s.repo.SetDownloadResult(ctx, id, destPath, existingTitle, existingDuration, existingThumbURL); err != nil {
						s.log.Error("set download result failed after reuse", "run_id", id, "err", err)
					}
					// Clear active_phase since download is already done.
					if err := s.repo.SetActivePhase(ctx, id, ""); err != nil {
						s.log.Error("clear active_phase failed after reuse", "run_id", id, "err", err)
					}

					s.log.Info("pipeline run reused video", "run_id", id, "url", normalized, "source", existingPath)
					// Emit video_info so the frontend can compute Segment Duration suggestions
					// from the actual video duration instead of showing hardcoded fallback values.
					s.hub.Publish(jobKey, hub.SSEEvent{
						Type: "video_info",
						Data: videoInfoPayload{
							RunID:        id,
							Title:        existingTitle,
							ThumbnailURL: existingThumbURL,
							DurationSec:  int(existingDuration),
						},
					})
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
		if req.ChannelID != nil {
			if err := s.repo.SetChannelID(ctx, id, *req.ChannelID); err != nil {
				s.log.Warn("failed to save channel_id on run", "run_id", id, "err", err)
			}
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
		go s.runExecution(id)
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
		if err := s.repo.AdvanceState(ctx, id, StateWaitingThumbnailConfig, StateWaitingReviewMetadata); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
					return AdvanceResult{State: curr.State}, nil
				}
			}
			return AdvanceResult{}, fmt.Errorf("advance run %d: %w", id, err)
		}
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "state_changed",
			Data: stateChangedPayload{RunID: id, State: StateWaitingReviewMetadata},
		})
		if s.metaGenerator != nil && s.metaGenerator.Available() {
			go s.metaGenerator.PrefillBatch(context.Background(), id)
		}
		return AdvanceResult{State: StateWaitingReviewMetadata}, nil

	case StateWaitingReviewMetadata:
		return s.advanceSimple(ctx, id, StateWaitingReviewMetadata, StateWaitingReviewClips)

	case StateWaitingReviewClips:
		return AdvanceResult{}, fmt.Errorf("use POST /gates/review-clips to advance from %s", StateWaitingReviewClips)

	case StateWaitingUploadConfirm:
		return s.confirmUpload(ctx, id, run)

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
	thumbnailURL := metaInfo.ThumbnailURL
	if thumbnailURL == "" {
		thumbnailURL = dlInfo.ThumbnailURL
	}
	if err := s.repo.SetDownloadResult(ctx, id, dlInfo.FilePath, title, durationSec, thumbnailURL); err != nil {
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

// runExecution is the background goroutine that runs the full transcription + highlight detection pipeline.
func (s *Service) runExecution(id int64) {
	jobKey := fmt.Sprintf("%d", id)
	ctx := context.Background()

	// Load run for video_path, url, duration_sec.
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("runExecution: failed to load run", "run_id", id, "err", err)
		_ = s.repo.Finish(ctx, id, StateError, fmt.Sprintf("load run: %s", err))
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "error",
			Data: map[string]interface{}{"run_id": id, "state": StateError, "message": fmt.Sprintf("load run: %s", err)},
		})
		return
	}

	req := processor.ExecRequest{
		RunID:       id,
		VideoPath:   run.VideoPath,
		VideoURL:    run.URL,
		DurationSec: run.DurationSec,
		DataDir:     s.dataDir,
		Mode:        run.Mode,

		PublishPhaseProgress: func(phase string, pct float64) {
			s.hub.Publish(jobKey, hub.SSEEvent{
				Type: "phase_progress",
				Data: phaseProgressPayload{RunID: id, Phase: phase, PercentDone: pct},
			})
		},
		PublishStateChange: func(state string) {
			s.hub.Publish(jobKey, hub.SSEEvent{
				Type: "state_changed",
				Data: stateChangedPayload{RunID: id, State: state},
			})
		},
		PublishError: func(msg string) {
			s.hub.Publish(jobKey, hub.SSEEvent{
				Type: "error",
				Data: map[string]interface{}{"run_id": id, "state": StateError, "message": msg},
			})
		},

		FindTranscriptByURL: s.repo.FindTranscriptByURL,
		SetTranscriptPath:   s.repo.SetTranscriptPath,
		AdvanceState:        s.repo.AdvanceState,
		Finish:              s.repo.Finish,
		ReplaceHighlights:   s.highlightRepo.ReplaceForRun,
	}

	if execErr := processor.RunExecution(ctx, req); execErr != nil {
		s.log.Error("runExecution failed", "run_id", id, "err", execErr)
		return
	}

	// For longform mode, automatically trigger clip generation (no highlight review gate).
	if run.Mode == "longform" {
		go s.runClipGeneration(id)
	}
}

// runClipGeneration generates clips for all selected highlights with the full effect pipeline.
func (s *Service) runClipGeneration(id int64) {
	jobKey := fmt.Sprintf("%d", id)
	runCtx, cancel := context.WithCancel(context.Background())
	s.cancelMap.Store(id, cancel)
	defer func() {
		cancel()
		s.cancelMap.Delete(id)
	}()
	ctx := runCtx
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

	// Check for clip reuse (skip_regenerate toggle from mode config)
	var reuseCfg struct {
		SkipRegenerate bool `json:"skip_regenerate"`
	}
	_ = json.Unmarshal([]byte(run.ModeConfigJSON), &reuseCfg)

	if reuseCfg.SkipRegenerate && run.URL != "" {
		_, priorRunID, findErr := s.repo.HasPriorRunWithClips(ctx, run.URL)
		if findErr != nil {
			log.Warn("prior run lookup failed, proceeding with fresh generation", "err", findErr)
		} else if priorRunID > 0 && priorRunID != id {
			priorClips, clipsErr := s.clipRepo.FindReadyClipsByRunID(ctx, priorRunID)
			if clipsErr != nil {
				log.Warn("prior clips lookup failed, proceeding with fresh generation", "err", clipsErr)
			} else if len(priorClips) > 0 && allFilesExist(priorClips) {
				if s.reuseClips(ctx, id, priorClips, log) {
					log.Info("reused clips from prior run", "source_run", priorRunID, "count", len(priorClips))
					s.hub.Publish(jobKey, hub.SSEEvent{
						Type: "phase_progress",
						Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: 100},
					})
					s.advanceClipGenDone(id)
					return
				}
				log.Warn("clip reuse failed, falling through to fresh generation")
			}
		}
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

	// Apply mode overrides so clip generation uses the same effect config as preview
	processor.ApplyModeOverrides(&channelCfg, json.RawMessage(run.ModeConfigJSON))
	blurEdgePct, noiseStrength := processor.ParseAntiDupExtras(json.RawMessage(run.ModeConfigJSON))

	// Resolve relative asset paths + logo avatar fallback (same as preview)
	overlayLibraryPath := ""
	if s.settingRepo != nil {
		overlayLibraryPath, _ = s.settingRepo.Get(ctx, "overlay_library_path")
	}
	channelID := int64(0)
	if run.ChannelID.Valid {
		channelID = run.ChannelID.Int64
	}
	processor.EnrichChannelConfig(&channelCfg, overlayLibraryPath, s.dataDir, channelID)

	// Longform mode: segment the video by time windows.
	if run.Mode == "longform" {
		var modeCfg struct {
			SegmentSec float64 `json:"segment_secs"`
		}
		if err := json.Unmarshal([]byte(run.ModeConfigJSON), &modeCfg); err != nil {
			log.Warn("failed to parse mode_config_json, using default segment_secs", "err", err)
		}
		segmentSec := modeCfg.SegmentSec
		if segmentSec <= 0 {
			log.Warn("segment_secs not set in mode_config, using default", "default_sec", 600)
			segmentSec = 600
		}
		if run.DurationSec <= 0 {
			log.Warn("duration_sec is 0, cannot segment — advancing without clips")
			s.advanceClipGenDone(id)
			return
		}
		segments := processor.ComputeSegments(float64(run.DurationSec), segmentSec)
		inputs := make([]clipInput, len(segments))
		for i, seg := range segments {
			inputs[i] = clipInput{
				startSec: seg.StartSec,
				endSec:   seg.EndSec,
				title:    "",
			}
		}
		s.generateClipsFromInputs(ctx, id, run, inputs, channelCfg, blurEdgePct, noiseStrength, publishError)
		return
	}

	// AI mode: one clip per selected highlight.
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

	inputs := make([]clipInput, 0, len(highlights))
	for _, hl := range highlights {
		startSec := hl.AdjStartSec
		endSec := hl.AdjEndSec
		if startSec == 0 && endSec == 0 {
			startSec = hl.StartSec
			endSec = hl.EndSec
		}
		inputs = append(inputs, clipInput{
			startSec:    startSec,
			endSec:      endSec,
			title:       hl.Text,
			highlightID: sql.NullInt64{Int64: hl.ID, Valid: true},
		})
	}
	s.generateClipsFromInputs(ctx, id, run, inputs, channelCfg, blurEdgePct, noiseStrength, publishError)
}

// clipInput describes a single clip to generate, used by both AI and longform modes.
type clipInput struct {
	startSec    float64
	endSec      float64
	title       string
	highlightID sql.NullInt64 // set for AI highlights, zero-value for longform
}

// segInfo holds the clip ID and duration for a segment during generation.
type segInfo struct {
	clipID int64
	dur    float64
}

// generateClipsFromInputs generates clips using a shared Phase A/B/C pipeline:
// Phase A: batch-insert all clip rows as pending
// Phase B: sequential stream-copy CutSegment → raw files
// Phase C: parallel effects via GenerateClip (including per-clip whisper for captions)
func (s *Service) generateClipsFromInputs(ctx context.Context, id int64, run *database.PipelineRun,
	inputs []clipInput, channelCfg database.ChannelConfig, blurEdgePct, noiseStrength float64, publishError func(string)) {

	log := s.log.With("run_id", id, "phase", "clip_gen")
	jobKey := fmt.Sprintf("%d", id)

	if len(inputs) == 0 {
		log.Warn("no clip inputs — advancing without clips")
		s.advanceClipGenDone(id)
		return
	}

	log.Info("starting clip generation", "total_clips", len(inputs))

	// Phase A: batch-insert all clip rows as pending, collect {clipID, dur}.
	infos := make([]segInfo, 0, len(inputs))
	for _, inp := range inputs {
		clip := &database.PipelineClip{
			RunID:       id,
			HighlightID: inp.highlightID,
			StartSec:    inp.startSec,
			EndSec:      inp.endSec,
			DurationSec: inp.endSec - inp.startSec,
			Title:       inp.title,
			IsSelected:  true,
		}
		clipID, err := s.clipRepo.Create(ctx, clip)
		if err != nil {
			publishError(fmt.Sprintf("create clip row failed: %s", err))
			return
		}
		infos = append(infos, segInfo{clipID: clipID, dur: inp.endSec - inp.startSec})
	}

	// Ensure output directory exists before Phase B.
	// Path pattern: clips/{videoID}_{totalClips}/{videoID}_clip_{N}.mp4
	videoID := downloader.ExtractVideoID(run.URL)
	if videoID == "" {
		videoID = fmt.Sprintf("run_%d", id)
	}
	clipsSubDir := fmt.Sprintf("%s_%d", videoID, len(inputs))
	clipsDir := filepath.Join(s.dataDir, "clips", clipsSubDir)
	if err := os.MkdirAll(clipsDir, 0o755); err != nil {
		publishError(fmt.Sprintf("create clips dir: %s", err))
		return
	}

	// Phase B: sequential stream copy → rawPath = {clipsDir}/{videoID}_clip_{N}_raw.mp4
	rawPaths := make([]string, len(inputs))
	for i, inp := range inputs {
		info := infos[i]
		rawPath := filepath.Join(clipsDir, fmt.Sprintf("%s_clip_%d_raw.mp4", videoID, i+1))

		err := processor.CutSegment(ctx, run.VideoPath, inp.startSec, inp.endSec-inp.startSec, rawPath, nil)
		if err != nil {
			log.Error("cut segment failed", "clip_id", info.clipID, "err", err)
			_ = s.clipRepo.UpdateStatus(ctx, info.clipID, "error")
		} else {
			rawPaths[i] = rawPath
		}

		// emit cumulative cut-phase 0–50% for the main progress bar
		pct := float64(i+1) / float64(len(inputs)) * 50
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "phase_progress",
			Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: pct},
		})
	}

	// Count valid raw paths.
	totalValid := 0
	for _, p := range rawPaths {
		if p != "" {
			totalValid++
		}
	}

	if totalValid == 0 {
		log.Warn("all stream copies failed, skipping effects phase")
		s.advanceClipGenDone(id)
		return
	}

	// Initialize per-clip progress at 0% so all bars appear before effects start.
	for i, info := range infos {
		if rawPaths[i] == "" {
			continue
		}
		cid := info.clipID
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "phase_progress",
			Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: 0, ClipID: &cid},
		})
	}

	// Phase C: parallel effects via GenerateClip.
	// Track per-clip progress so the overall bar reflects the average.
	var wg sync.WaitGroup
	var mu sync.Mutex
	clipPcts := make(map[int64]float64, totalValid)

	emitOverall := func() {
		// must be called with mu held
		var sum float64
		for _, p := range clipPcts {
			sum += p
		}
		overall := 50 + (sum/float64(totalValid))/100*50
		s.hub.Publish(jobKey, hub.SSEEvent{
			Type: "phase_progress",
			Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: overall},
		})
	}

	for i, info := range infos {
		rawPath := rawPaths[i]
		if rawPath == "" {
			continue
		}
		clipNum := i + 1
		wg.Add(1)
		go func(clipInfo segInfo, rawPath string, clipNum int) {
			defer wg.Done()

			clipID := clipInfo.clipID
			dur := clipInfo.dur
			outPath := filepath.Join(clipsDir, fmt.Sprintf("%s_clip_%d.mp4", videoID, clipNum))

			clipReq := processor.ClipRequest{
				RunID:          id,
				ClipID:         clipID,
				VideoPath:      rawPath,
				StartSec:       0,
				EndSec:         dur,
				ChannelCfg:     channelCfg,
				ModeCfg:        json.RawMessage(run.ModeConfigJSON),
				DataDir:        s.dataDir,
				BlurEdgePct:    blurEdgePct,
				NoiseStrength:  noiseStrength,
				TranscriptPath: "",
				OutputPath:     outPath,
				OnProgress: func(pct float64) {
					cid := clipID
					s.hub.Publish(jobKey, hub.SSEEvent{
						Type: "phase_progress",
						Data: phaseProgressPayload{RunID: id, Phase: "cut", PercentDone: pct, ClipID: &cid},
					})
					mu.Lock()
					clipPcts[clipID] = pct
					emitOverall()
					mu.Unlock()
				},
			}

			finalPath, err := processor.GenerateClip(ctx, clipReq)
			if err != nil {
				log.Error("effects generation failed", "clip_id", clipID, "err", err)
				_ = s.clipRepo.UpdateStatus(ctx, clipID, "error")
			} else {
				_ = s.clipRepo.UpdateFilePath(ctx, clipID, finalPath)
				_ = s.clipRepo.UpdateStatus(ctx, clipID, "ready")
				_ = os.Remove(rawPath)
			}

			mu.Lock()
			clipPcts[clipID] = 100
			emitOverall()
			mu.Unlock()
		}(info, rawPath, clipNum)
	}

	wg.Wait()

	// Publish 100% and advance state.
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

// allFilesExist returns true if every clip's FilePath exists on disk.
func allFilesExist(clips []database.PipelineClip) bool {
	for _, c := range clips {
		if _, err := os.Stat(c.FilePath); err != nil {
			return false
		}
	}
	return true
}

// reuseClips reuses clip files from a prior run by pointing new DB rows to existing files.
// The files already live in clips/{videoID}_{N}/ so no copy is needed — just insert DB rows.
// Returns true on success, false if any step fails (caller should fall through to fresh generation).
func (s *Service) reuseClips(ctx context.Context, runID int64, clips []database.PipelineClip, log *slog.Logger) bool {
	for i, src := range clips {
		newClip := &database.PipelineClip{
			RunID:          runID,
			StartSec:       src.StartSec,
			EndSec:         src.EndSec,
			DurationSec:    src.DurationSec,
			Title:          src.Title,
			Description:    src.Description,
			Tags:           src.Tags,
			ThumbnailStyle: src.ThumbnailStyle,
			IsSelected:     src.IsSelected,
		}
		newClipID, err := s.clipRepo.Create(ctx, newClip)
		if err != nil {
			log.Error("insert reused clip row", "src_clip", src.ID, "err", err)
			return false
		}

		// Point to existing file — no copy needed since path is video-based, not run-based
		if err := s.clipRepo.UpdateFilePath(ctx, newClipID, src.FilePath); err != nil {
			log.Error("update reused clip file_path", "err", err)
			return false
		}
		if err := s.clipRepo.UpdateStatus(ctx, newClipID, "ready"); err != nil {
			log.Error("update reused clip status", "err", err)
			return false
		}

		// Reuse thumbnail path if it exists
		if src.ThumbnailPath != "" {
			if _, statErr := os.Stat(src.ThumbnailPath); statErr == nil {
				_ = s.clipRepo.UpdateThumbnailPath(ctx, newClipID, src.ThumbnailPath, src.ThumbnailStyle)
			}
		}

		_ = i // suppress unused warning
	}
	return true
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

// confirmUpload queues all selected clips for the run to the upload_queue directory,
// inserts upload records, then advances the run to DONE.
func (s *Service) confirmUpload(ctx context.Context, id int64, run *database.PipelineRun) (AdvanceResult, error) {
	jobKey := fmt.Sprintf("%d", id)

	if !run.ChannelID.Valid {
		return AdvanceResult{}, fmt.Errorf("run %d has no channel_id set", id)
	}
	channelID := run.ChannelID.Int64

	// Load channel config for metadata
	cfg, err := s.channelCfgRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return AdvanceResult{}, fmt.Errorf("load channel config for channel %d: %w", channelID, err)
	}

	// Load selected clips
	allClips, err := s.clipRepo.ListByRun(ctx, id)
	if err != nil {
		return AdvanceResult{}, fmt.Errorf("load clips for run %d: %w", id, err)
	}
	var selected []database.PipelineClip
	for _, c := range allClips {
		if c.IsSelected {
			selected = append(selected, c)
		}
	}
	if len(selected) == 0 {
		return AdvanceResult{}, fmt.Errorf("no selected clips to queue for run %d", id)
	}

	// Queue each clip
	for i, clip := range selected {
		meta := uploader.BuildMetadata(clip, *cfg)
		metaBytes, _ := json.Marshal(meta)

		qVideo, qThumb, saveErr := s.queueStorage.SaveToQueue(clip.FilePath, clip.ThumbnailPath, meta)
		if saveErr != nil {
			s.log.Error("save to queue failed", "clip_id", clip.ID, "err", saveErr)
			return AdvanceResult{}, fmt.Errorf("save clip %d to queue: %w", clip.ID, saveErr)
		}

		_, createErr := s.uploadRepo.CreateQueued(ctx, channelID, clip.ID, qVideo, qThumb, string(metaBytes), "{}", i+1)
		if createErr != nil {
			s.log.Error("create queued upload failed", "clip_id", clip.ID, "err", createErr)
			return AdvanceResult{}, fmt.Errorf("create upload record for clip %d: %w", clip.ID, createErr)
		}
	}

	// Advance run directly to DONE (skip UPLOADING — we're queuing, not immediately uploading)
	if err := s.repo.AdvanceState(ctx, id, StateWaitingUploadConfirm, StateDone); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if curr, _ := s.repo.GetByID(ctx, id); curr != nil {
				return AdvanceResult{State: curr.State}, nil
			}
		}
		return AdvanceResult{}, fmt.Errorf("advance run %d to done: %w", id, err)
	}
	s.hub.Publish(jobKey, hub.SSEEvent{
		Type: "state_changed",
		Data: stateChangedPayload{RunID: id, State: StateDone},
	})
	s.log.Info("upload confirm: clips queued", "run_id", id, "count", len(selected))
	return AdvanceResult{State: StateDone}, nil
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
