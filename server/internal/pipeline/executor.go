package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/effects"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// Executor runs the compute phases for EXECUTING, GENERATING_CLIPS, and UPLOADING states.
// Each public method is intended to be called in a goroutine from the HTTP handler.
type Executor struct {
	repo        *Repo
	hub         *hub.SSEHub
	downloader  Downloader
	cutter      Cutter
	transcriber Transcriber
	analyzer    Analyzer
	shorts      ShortsGenerator
	thumbnail   ThumbnailMaker
	uploader    Uploader
	effects     *effects.EffectsService  // nil = anti-dup disabled
	chanCfg     ChannelConfigReader       // nil = config unavailable
	log         *slog.Logger
}

// NewExecutor constructs an Executor. Nil subsystem adapters cause that phase
// to be skipped with a warning.
func NewExecutor(
	repo *Repo,
	h *hub.SSEHub,
	downloader Downloader,
	cutter Cutter,
	transcriber Transcriber,
	analyzer Analyzer,
	shorts ShortsGenerator,
	thumbnail ThumbnailMaker,
	uploader Uploader,
	effectsSvc *effects.EffectsService,
	chanCfg ChannelConfigReader,
) *Executor {
	return &Executor{
		repo:        repo,
		hub:         h,
		downloader:  downloader,
		cutter:      cutter,
		transcriber: transcriber,
		analyzer:    analyzer,
		shorts:      shorts,
		thumbnail:   thumbnail,
		uploader:    uploader,
		effects:     effectsSvc,
		chanCfg:     chanCfg,
		log:         slog.With("component", "pipeline.executor"),
	}
}

// ── EXECUTING phase ───────────────────────────────────────────────────────────

// RunExecuting runs the EXECUTING compute state.
// For AI mode: download → transcript → analyze → gate WAITING_REVIEW_HIGHLIGHTS
// For longform mode: download → transcript → segment → gate WAITING_REVIEW_METADATA
func (e *Executor) RunExecuting(ctx context.Context, runID int64) {
	logger := e.log.With("runID", runID, "phase", "executing")

	run, err := e.repo.GetRun(ctx, runID)
	if err != nil {
		logger.Error("load run failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}

	outputDir, err := resolveOutputDir(runID)
	if err != nil {
		logger.Error("resolve output dir failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		logger.Error("create output dir failed", "dir", outputDir, "err", err)
		e.failRun(ctx, runID, err)
		return
	}

	// ── Download ──────────────────────────────────────────────────────────────
	if err := e.repo.UpdateRunState(ctx, runID, StateExecuting, string(PhaseDownload)); err != nil {
		logger.Error("update state failed", "err", err)
	}
	publishPhaseProgress(e.hub, runID, PhaseDownload, 0, nil)

	videoPath, err := e.download(ctx, run.URL, outputDir)
	if err != nil {
		logger.Error("download failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}
	publishPhaseProgress(e.hub, runID, PhaseDownload, 100, nil)

	// ── Transcript ────────────────────────────────────────────────────────────
	if err := e.repo.UpdateRunState(ctx, runID, StateExecuting, string(PhaseTranscript)); err != nil {
		logger.Error("update state failed", "err", err)
	}
	publishPhaseProgress(e.hub, runID, PhaseTranscript, 0, nil)

	transcriptPath, err := e.transcribe(ctx, videoPath)
	if err != nil {
		logger.Error("transcription failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}
	publishPhaseProgress(e.hub, runID, PhaseTranscript, 100, nil)

	if err := e.repo.UpdateRunPaths(ctx, runID, videoPath, transcriptPath); err != nil {
		logger.Error("update paths failed", "err", err)
	}

	switch run.Mode {
	case WorkflowAI:
		// ── AI analyze ────────────────────────────────────────────────────────
		if err := e.repo.UpdateRunState(ctx, runID, StateExecuting, string(PhaseAnalyze)); err != nil {
			logger.Error("update state failed", "err", err)
		}
		publishPhaseProgress(e.hub, runID, PhaseAnalyze, 0, nil)

		highlights, err := e.analyze(ctx, transcriptPath)
		if err != nil {
			logger.Error("analysis failed", "err", err)
			e.failRun(ctx, runID, err)
			return
		}
		publishPhaseProgress(e.hub, runID, PhaseAnalyze, 100, nil)

		if err := e.repo.InsertHighlights(ctx, runID, highlights); err != nil {
			logger.Error("insert highlights failed", "err", err)
			e.failRun(ctx, runID, err)
			return
		}

		// Transition → WAITING_REVIEW_HIGHLIGHTS
		next, err := Advance(StateExecuting, WorkflowAI, EventExecutionDone)
		if err != nil {
			logger.Error("state machine advance failed", "err", err)
			e.failRun(ctx, runID, err)
			return
		}
		if err := e.repo.UpdateRunState(ctx, runID, next, ""); err != nil {
			logger.Error("update state failed", "err", err)
		}
		publishStateChanged(e.hub, runID, next)
		publishGateOpened(e.hub, runID, next)

	case WorkflowLongform:
		// ── Longform segment ──────────────────────────────────────────────────
		cfg := parseModeConfig(run.ModeConfigJSON)
		segmentSecs := cfg.SegmentSecs
		if segmentSecs <= 0 {
			segmentSecs = 60
		}
		segments, err := buildLongformSegments(transcriptPath, segmentSecs)
		if err != nil {
			logger.Error("segment failed", "err", err)
			e.failRun(ctx, runID, err)
			return
		}
		// Create placeholder clips for each segment
		for i, seg := range segments {
			c := &Clip{
				RunID:        runID,
				FilePath:     "",
				StartSec:     seg.StartSec,
				EndSec:       seg.EndSec,
				DurationSec:  seg.EndSec - seg.StartSec,
				IsSelected:   true,
				UploadStatus: "pending",
				Title:        fmt.Sprintf("Clip %d", i+1),
			}
			if _, err := e.repo.InsertClip(ctx, c); err != nil {
				logger.Error("insert clip failed", "i", i, "err", err)
			}
		}

		// Transition → WAITING_REVIEW_METADATA
		next, err := Advance(StateExecuting, WorkflowLongform, EventExecutionDone)
		if err != nil {
			logger.Error("state machine advance failed", "err", err)
			e.failRun(ctx, runID, err)
			return
		}
		if err := e.repo.UpdateRunState(ctx, runID, next, ""); err != nil {
			logger.Error("update state failed", "err", err)
		}
		publishStateChanged(e.hub, runID, next)
		publishGateOpened(e.hub, runID, next)

	default:
		e.failRun(ctx, runID, fmt.Errorf("unknown workflow mode: %s", run.Mode))
	}
}

// ── GENERATING_CLIPS phase ────────────────────────────────────────────────────

// RunGeneratingClips runs the GENERATING_CLIPS compute state.
// cut → generate shorts → generate thumbnails → gate WAITING_THUMBNAIL_CONFIG
func (e *Executor) RunGeneratingClips(ctx context.Context, runID int64) {
	logger := e.log.With("runID", runID, "phase", "generating_clips")

	run, err := e.repo.GetRun(ctx, runID)
	if err != nil {
		logger.Error("load run failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}

	highlights, err := e.repo.GetHighlights(ctx, runID)
	if err != nil {
		logger.Error("get highlights failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}

	outputDir, err := resolveOutputDir(runID)
	if err != nil {
		logger.Error("resolve output dir failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}

	selected := make([]Highlight, 0, len(highlights))
	for _, h := range highlights {
		if h.IsSelected {
			selected = append(selected, h)
		}
	}

	segments := make([]SegmentRange, len(selected))
	for i, h := range selected {
		segments[i] = SegmentRange{StartSec: h.AdjStartSec, EndSec: h.AdjEndSec}
	}

	// ── Cut ───────────────────────────────────────────────────────────────────
	if err := e.repo.UpdateRunState(ctx, runID, StateGeneratingClips, string(PhaseCut)); err != nil {
		logger.Error("update state failed", "err", err)
	}
	publishPhaseProgress(e.hub, runID, PhaseCut, 0, nil)

	clipPaths, err := e.cut(ctx, run.VideoPath, segments, outputDir)
	if err != nil {
		logger.Error("cut failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}
	publishPhaseProgress(e.hub, runID, PhaseCut, 100, nil)

	// ── Shorts ────────────────────────────────────────────────────────────────
	if err := e.repo.UpdateRunState(ctx, runID, StateGeneratingClips, string(PhaseShorts)); err != nil {
		logger.Error("update state failed", "err", err)
	}
	publishPhaseProgress(e.hub, runID, PhaseShorts, 0, nil)

	shortPaths, _ := e.generateShorts(ctx, run.VideoPath, segments, outputDir)
	publishPhaseProgress(e.hub, runID, PhaseShorts, 100, nil)
	_ = shortPaths

	// ── Thumbnails ────────────────────────────────────────────────────────────
	if err := e.repo.UpdateRunState(ctx, runID, StateGeneratingClips, string(PhaseThumbnail)); err != nil {
		logger.Error("update state failed", "err", err)
	}

	for i, clipPath := range clipPaths {
		thumbPath := ""
		if e.thumbnail != nil {
			tp, tErr := e.thumbnail.MakeThumbnail(ctx, clipPath, "default")
			if tErr != nil {
				logger.Warn("thumbnail failed for clip", "i", i, "err", tErr)
			} else {
				thumbPath = tp
			}
		}

		var hlID *int64
		if i < len(selected) {
			v := selected[i].ID
			hlID = &v
		}
		c := &Clip{
			RunID:         runID,
			HighlightID:   hlID,
			FilePath:      clipPath,
			ThumbnailPath: thumbPath,
			IsSelected:    true,
			UploadStatus:  "pending",
		}
		if i < len(selected) {
			c.StartSec = selected[i].AdjStartSec
			c.EndSec = selected[i].AdjEndSec
			c.DurationSec = selected[i].AdjEndSec - selected[i].AdjStartSec
		}
		if _, err := e.repo.InsertClip(ctx, c); err != nil {
			logger.Error("insert clip failed", "i", i, "err", err)
		}
		publishPhaseProgress(e.hub, runID, PhaseThumbnail, float64(i+1)/float64(len(clipPaths))*100, nil)
	}

	// Transition → WAITING_THUMBNAIL_CONFIG
	next, err := Advance(StateGeneratingClips, run.Mode, EventClipsGenerated)
	if err != nil {
		logger.Error("state machine advance failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}
	if err := e.repo.UpdateRunState(ctx, runID, next, ""); err != nil {
		logger.Error("update state failed", "err", err)
	}
	publishStateChanged(e.hub, runID, next)
	publishGateOpened(e.hub, runID, next)
}

// ── UPLOADING phase ───────────────────────────────────────────────────────────

// RunUploading runs the UPLOADING compute state.
// Uploads each selected clip sequentially, publishes per-clip progress via SSE.
func (e *Executor) RunUploading(ctx context.Context, runID int64, privacy string) {
	logger := e.log.With("runID", runID, "phase", "uploading")

	clips, err := e.repo.GetClips(ctx, runID)
	if err != nil {
		logger.Error("get clips failed", "err", err)
		e.failRun(ctx, runID, err)
		return
	}

	selected := make([]Clip, 0, len(clips))
	for _, c := range clips {
		if c.IsSelected {
			selected = append(selected, c)
		}
	}

	if e.uploader == nil {
		logger.Warn("uploader not configured, skipping upload")
		e.completeRun(ctx, runID)
		return
	}

	for i, c := range selected {
		select {
		case <-ctx.Done():
			e.failRun(ctx, runID, ctx.Err())
			return
		default:
		}

		if err := e.repo.UpdateClipUpload(ctx, c.ID, "uploading", ""); err != nil {
			logger.Error("update clip upload status failed", "clipID", c.ID, "err", err)
		}
		publishPhaseProgress(e.hub, runID, PhaseUpload, float64(i)/float64(len(selected))*100,
			map[string]interface{}{"clip_id": c.ID, "clip_index": i})

		youtubeID, err := e.uploader.UploadClip(ctx, UploadClipRequest{
			FilePath:      c.FilePath,
			ThumbnailPath: c.ThumbnailPath,
			Title:         c.Title,
			Description:   c.Description,
			Tags:          strings.Split(c.Tags, ","),
			Privacy:       privacy,
		})
		if err != nil {
			logger.Error("upload failed", "clipID", c.ID, "err", err)
			_ = e.repo.UpdateClipUpload(ctx, c.ID, "error", "")
			continue
		}

		if err := e.repo.UpdateClipUpload(ctx, c.ID, "done", youtubeID); err != nil {
			logger.Error("update clip upload result failed", "clipID", c.ID, "err", err)
		}
		publishPhaseProgress(e.hub, runID, PhaseUpload, float64(i+1)/float64(len(selected))*100,
			map[string]interface{}{"clip_id": c.ID, "youtube_id": youtubeID})
	}

	e.completeRun(ctx, runID)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (e *Executor) download(ctx context.Context, url, outputDir string) (string, error) {
	if e.downloader == nil {
		return "", fmt.Errorf("downloader not configured")
	}
	return e.downloader.DownloadVideo(ctx, url, outputDir)
}

func (e *Executor) transcribe(ctx context.Context, videoPath string) (string, error) {
	if e.transcriber == nil {
		e.log.Warn("transcriber not configured, skipping")
		return "", nil
	}
	return e.transcriber.Transcribe(ctx, videoPath)
}

func (e *Executor) analyze(ctx context.Context, transcriptPath string) ([]Highlight, error) {
	if e.analyzer == nil {
		return nil, fmt.Errorf("analyzer not configured")
	}
	return e.analyzer.Analyze(ctx, transcriptPath)
}

func (e *Executor) cut(ctx context.Context, videoPath string, segments []SegmentRange, outputDir string) ([]string, error) {
	if e.cutter == nil {
		e.log.Warn("cutter not configured, returning source video")
		return []string{videoPath}, nil
	}
	return e.cutter.CutSegments(ctx, videoPath, segments, outputDir)
}

func (e *Executor) generateShorts(ctx context.Context, videoPath string, segments []SegmentRange, outputDir string) ([]string, error) {
	if e.shorts == nil {
		e.log.Warn("shorts generator not configured, skipping")
		return nil, nil
	}
	return e.shorts.GenerateShorts(ctx, videoPath, segments, outputDir)
}

func (e *Executor) failRun(ctx context.Context, runID int64, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	if finishErr := e.repo.FinishRun(ctx, runID, StateError, msg); finishErr != nil {
		e.log.Error("finish run (error) failed", "runID", runID, "err", finishErr)
	}
	publishError(e.hub, runID, msg)
}

func (e *Executor) completeRun(ctx context.Context, runID int64) {
	next, _ := Advance(StateUploading, "", EventUploadDone)
	if err := e.repo.FinishRun(ctx, runID, next, ""); err != nil {
		e.log.Error("finish run failed", "runID", runID, "err", err)
	}
	publishDone(e.hub, runID)
}

// resolveOutputDir returns the base output dir for a run.
func resolveOutputDir(runID int64) (string, error) {
	base := os.Getenv("AUTOCUT_DIR")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		base = filepath.Join(home, ".autocut")
	}
	return filepath.Join(base, "pipeline", fmt.Sprintf("%d", runID)), nil
}

// parseModeConfig safely parses JSON into ModeConfig.
func parseModeConfig(modeConfigJSON string) ModeConfig {
	var cfg ModeConfig
	if modeConfigJSON != "" && modeConfigJSON != "{}" {
		_ = json.Unmarshal([]byte(modeConfigJSON), &cfg)
	}
	return cfg
}

// buildLongformSegments reads the transcript and divides it into fixed-duration segments.
func buildLongformSegments(transcriptPath string, segmentSecs int) ([]SegmentRange, error) {
	if transcriptPath == "" {
		return []SegmentRange{{StartSec: 0, EndSec: float64(segmentSecs)}}, nil
	}
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		return nil, fmt.Errorf("read transcript: %w", err)
	}
	var t struct {
		Segments []struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(data, &t); err != nil || len(t.Segments) == 0 {
		return []SegmentRange{{StartSec: 0, EndSec: float64(segmentSecs)}}, nil
	}

	totalEnd := t.Segments[len(t.Segments)-1].End
	var segs []SegmentRange
	start := 0.0
	dur := float64(segmentSecs)
	for start < totalEnd {
		end := start + dur
		if end > totalEnd {
			end = totalEnd
		}
		segs = append(segs, SegmentRange{StartSec: start, EndSec: end})
		start = end
	}
	return segs, nil
}

// UploadProgress mirrors uploader.UploadProgress — re-declared here to avoid
// import cycles. The actual channel type from uploader.YouTubeUploader returns
// uploader.UploadProgress. Our adapter already decouples this.
type uploadProgressInternal struct {
	Done    bool
	VideoID string
	Err     error
}

// uploadProgressFrom avoids importing uploader.UploadProgress at the types layer.
// (The UploaderAdapter in adapters.go handles the concrete type translation.)
// This stub is here to satisfy the compiler for future test mocks — no runtime use.
var _ = time.Second // keep import used
