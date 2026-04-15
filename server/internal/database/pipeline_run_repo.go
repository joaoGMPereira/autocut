package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// PipelineRunRepo provides CRUD for pipeline_runs (v9 schema).
// v9 uses state/active_phase instead of status/current_step.
type PipelineRunRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewPipelineRunRepo(db *sql.DB, log *slog.Logger) *PipelineRunRepo {
	return &PipelineRunRepo{db: db, log: log.With("repo", "pipeline_run")}
}

const pipelineRunCols = `id, channel_id, url, mode, state, active_phase, error, mode_config_json, video_path, video_title, duration_sec, transcript_path, started_at, finished_at`

// Create inserts a new pipeline run and returns its ID.
func (r *PipelineRunRepo) Create(ctx context.Context, p *PipelineRun) (int64, error) {
	now := time.Now().UnixMilli()
	state := p.State
	if state == "" {
		state = "WAITING_URL"
	}
	modeConfig := p.ModeConfigJSON
	if modeConfig == "" {
		modeConfig = "{}"
	}
	mode := p.Mode
	if mode == "" {
		mode = "ai"
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pipeline_runs (channel_id, url, mode, state, active_phase, error, mode_config_json, video_path, transcript_path, started_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ChannelID, p.URL, mode, state, p.ActivePhase, p.Error, modeConfig, p.VideoPath, p.TranscriptPath, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert pipeline_run: %w", err)
	}
	return res.LastInsertId()
}

// GetByID returns a single pipeline run by ID.
func (r *PipelineRunRepo) GetByID(ctx context.Context, id int64) (*PipelineRun, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+pipelineRunCols+" FROM pipeline_runs WHERE id = ?", id)
	return scanPipelineRun(row)
}

// ListRunsFilter holds optional filters for ListAll.
type ListRunsFilter struct {
	ChannelID *int64
	State     string
	DateFrom  int64
	DateTo    int64
}

// ListAll returns paginated pipeline runs with optional filters.
func (r *PipelineRunRepo) ListAll(ctx context.Context, limit, offset int, f ListRunsFilter) ([]PipelineRun, int, error) {
	var whereClauses []string
	var filterArgs []interface{}

	if f.ChannelID != nil {
		whereClauses = append(whereClauses, "channel_id = ?")
		filterArgs = append(filterArgs, *f.ChannelID)
	}
	if f.State != "" {
		whereClauses = append(whereClauses, "state = ?")
		filterArgs = append(filterArgs, f.State)
	}
	if f.DateFrom != 0 {
		whereClauses = append(whereClauses, "started_at >= ?")
		filterArgs = append(filterArgs, f.DateFrom)
	}
	if f.DateTo != 0 {
		whereClauses = append(whereClauses, "started_at <= ?")
		filterArgs = append(filterArgs, f.DateTo)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pipeline_runs"+whereSQL, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pipeline_runs: %w", err)
	}

	dataArgs := append(filterArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+pipelineRunCols+" FROM pipeline_runs"+whereSQL+" ORDER BY started_at DESC LIMIT ? OFFSET ?",
		dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pipeline_runs: %w", err)
	}
	defer rows.Close()

	var result []PipelineRun
	for rows.Next() {
		p, err := scanPipelineRunRows(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, *p)
	}
	return result, total, rows.Err()
}

// StartDownload atomically sets url + active_phase='download' while keeping state=WAITING_URL.
// The run state is NOT advanced to EXECUTING here; that happens only after download completes.
func (r *PipelineRunRepo) StartDownload(ctx context.Context, id int64, url string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET url = ?, active_phase = 'download'
		 WHERE id = ? AND state = 'WAITING_URL'`,
		url, id,
	)
	if err != nil {
		return fmt.Errorf("start download pipeline_run %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SetActivePhase updates the active_phase field.
func (r *PipelineRunRepo) SetActivePhase(ctx context.Context, id int64, phase string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET active_phase = ? WHERE id = ?", phase, id)
	if err != nil {
		return fmt.Errorf("set active_phase pipeline_run %d: %w", id, err)
	}
	return nil
}

// SetVideoPath stores the downloaded video file path.
func (r *PipelineRunRepo) SetVideoPath(ctx context.Context, id int64, path string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET video_path = ? WHERE id = ?", path, id)
	if err != nil {
		return fmt.Errorf("set video_path pipeline_run %d: %w", id, err)
	}
	return nil
}

// SetChannelID updates the channel_id for a pipeline run.
func (r *PipelineRunRepo) SetChannelID(ctx context.Context, id, channelID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET channel_id = ? WHERE id = ?`,
		channelID, id)
	return err
}

// SetDownloadResult atomically persists video_path, video_title, and duration_sec
// after a successful download. All three fields must be non-empty/non-zero — callers
// must verify this before transitioning to WAITING_MODE.
func (r *PipelineRunRepo) SetDownloadResult(ctx context.Context, id int64, path, title string, durationSec int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET video_path = ?, video_title = ?, duration_sec = ? WHERE id = ?",
		path, title, durationSec, id,
	)
	if err != nil {
		return fmt.Errorf("set download result pipeline_run %d: %w", id, err)
	}
	return nil
}

// FindTranscriptByURL returns the transcript_path from the most recent run with the given URL
// that has a non-empty transcript_path. Returns ("", nil) if none found.
func (r *PipelineRunRepo) FindTranscriptByURL(ctx context.Context, url string) (string, error) {
	var path string
	err := r.db.QueryRowContext(ctx,
		`SELECT transcript_path FROM pipeline_runs WHERE url = ? AND transcript_path != '' ORDER BY id DESC LIMIT 1`,
		url,
	).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find transcript by url: %w", err)
	}
	return path, nil
}

// FindExistingVideo finds a prior run with the same URL that has a non-empty video_path.
// Returns the video_path, video_title, and duration_sec if found.
func (r *PipelineRunRepo) FindExistingVideo(ctx context.Context, url string) (path, title string, durationSec int64, found bool, err error) {
	err = r.db.QueryRowContext(ctx,
		`SELECT video_path, video_title, duration_sec FROM pipeline_runs WHERE url = ? AND video_path != '' ORDER BY id DESC LIMIT 1`,
		url,
	).Scan(&path, &title, &durationSec)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", 0, false, nil
	}
	if err != nil {
		return "", "", 0, false, fmt.Errorf("find existing video: %w", err)
	}
	return path, title, durationSec, true, nil
}

// StartDownloadAndAdvance atomically sets url, state=WAITING_MODE, active_phase='download'.
// Used when no prior video is found and a fresh download is needed.
func (r *PipelineRunRepo) StartDownloadAndAdvance(ctx context.Context, id int64, url string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET url = ?, state = 'WAITING_MODE', active_phase = 'download'
		 WHERE id = ? AND state = 'WAITING_URL'`,
		url, id,
	)
	if err != nil {
		return fmt.Errorf("start download and advance pipeline_run %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AdvanceState atomically transitions a run from fromState to toState.
// Returns sql.ErrNoRows if no row was updated (idempotency signal — run already moved on).
func (r *PipelineRunRepo) AdvanceState(ctx context.Context, id int64, fromState, toState string) error {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET state = ?, active_phase = '', finished_at = ?
		 WHERE id = ? AND state = ?`,
		toState, now, id, fromState,
	)
	if err != nil {
		return fmt.Errorf("advance state pipeline_run %d (%s→%s): %w", id, fromState, toState, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// StoreModeConfig persists the submitted mode and mode_config_json for a
// pipeline run. Must be called while the run is still in WAITING_MODE state.
func (r *PipelineRunRepo) StoreModeConfig(ctx context.Context, id int64, mode string, configJSON []byte) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET mode = ?, mode_config_json = ? WHERE id = ? AND state = 'WAITING_MODE'`,
		mode, string(configJSON), id,
	)
	if err != nil {
		return fmt.Errorf("store mode config run %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store mode config rows affected run %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("run %d not in WAITING_MODE state", id)
	}
	slog.Info("mode config stored", "run_id", id, "mode", mode)
	return nil
}

// HasPriorDoneRun checks if a prior completed run exists for the given URL.
// Returns (true, runID) if found, (false, 0) otherwise.
func (r *PipelineRunRepo) HasPriorDoneRun(ctx context.Context, url string) (bool, int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM pipeline_runs WHERE url = ? AND state = 'DONE' ORDER BY id DESC LIMIT 1`,
		url,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, fmt.Errorf("has prior done run: %w", err)
	}
	return true, id, nil
}

// Finish transitions the run to a terminal state.
func (r *PipelineRunRepo) Finish(ctx context.Context, id int64, state string, errMsg string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET state = ?, error = ?, active_phase = '', finished_at = ? WHERE id = ?",
		state, errMsg, now, id,
	)
	if err != nil {
		return fmt.Errorf("finish pipeline_run %d: %w", id, err)
	}
	return nil
}

// Cancel transitions a run to CANCELLED if not already terminal.
func (r *PipelineRunRepo) Cancel(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET state = 'CANCELLED', finished_at = ?
		 WHERE id = ? AND state NOT IN ('DONE','ERROR','CANCELLED')`,
		time.Now().UnixMilli(), id,
	)
	if err != nil {
		return fmt.Errorf("cancel pipeline_run %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes a pipeline run.
func (r *PipelineRunRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM pipeline_runs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete pipeline_run %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanPipelineRun(row *sql.Row) (*PipelineRun, error) {
	var p PipelineRun
	err := row.Scan(&p.ID, &p.ChannelID, &p.URL, &p.Mode, &p.State, &p.ActivePhase, &p.Error, &p.ModeConfigJSON, &p.VideoPath, &p.VideoTitle, &p.DurationSec, &p.TranscriptPath, &p.StartedAt, &p.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPipelineRunRows(rows *sql.Rows) (*PipelineRun, error) {
	var p PipelineRun
	err := rows.Scan(&p.ID, &p.ChannelID, &p.URL, &p.Mode, &p.State, &p.ActivePhase, &p.Error, &p.ModeConfigJSON, &p.VideoPath, &p.VideoTitle, &p.DurationSec, &p.TranscriptPath, &p.StartedAt, &p.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
