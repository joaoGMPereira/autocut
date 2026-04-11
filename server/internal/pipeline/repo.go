package pipeline

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Repo provides CRUD for pipeline_runs, pipeline_highlights, pipeline_clips.
type Repo struct {
	db  *sql.DB
	log *slog.Logger
}

// NewRepo creates a Repo.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db, log: slog.With("component", "pipeline.repo")}
}

// ── pipeline_runs ─────────────────────────────────────────────────────────────

const runCols = `id, channel_id, url, mode, state, active_phase, error, mode_config_json, video_path, transcript_path, started_at, finished_at`

// CreateRun inserts a new pipeline run and returns its ID.
func (r *Repo) CreateRun(ctx context.Context, channelID *int64, url string, state RunState) (int64, error) {
	now := time.Now().UnixMilli()
	var chID interface{} = nil
	if channelID != nil {
		chID = *channelID
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pipeline_runs (channel_id, url, mode, state, active_phase, error, mode_config_json, video_path, transcript_path, started_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chID, url, string(WorkflowAI), string(state), "", "", "{}", "", "", now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert pipeline_run: %w", err)
	}
	return res.LastInsertId()
}

// GetRun returns a run by ID.
func (r *Repo) GetRun(ctx context.Context, id int64) (*Run, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+runCols+" FROM pipeline_runs WHERE id = ?", id)
	return scanRun(row)
}

// ListRuns returns recent runs with optional channel filter.
func (r *Repo) ListRuns(ctx context.Context, limit, offset int, channelID *int64) ([]*Run, int64, error) {
	args := []interface{}{}
	where := ""
	if channelID != nil {
		where = "WHERE channel_id = ?"
		args = append(args, *channelID)
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM pipeline_runs " + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pipeline_runs: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+runCols+" FROM pipeline_runs "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pipeline_runs: %w", err)
	}
	defer rows.Close()

	var runs []*Run
	for rows.Next() {
		p, err := scanRunRow(rows)
		if err != nil {
			return nil, 0, err
		}
		runs = append(runs, p)
	}
	return runs, total, rows.Err()
}

// UpdateRunState updates the state and active_phase of a run.
func (r *Repo) UpdateRunState(ctx context.Context, id int64, state RunState, phase string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET state = ?, active_phase = ? WHERE id = ?`,
		string(state), phase, id)
	if err != nil {
		return fmt.Errorf("update run state %d: %w", id, err)
	}
	return nil
}

// UpdateRunMode stores the mode and mode_config_json after WAITING_MODE gate.
func (r *Repo) UpdateRunMode(ctx context.Context, id int64, mode WorkflowMode, modeConfigJSON string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET mode = ?, mode_config_json = ? WHERE id = ?`,
		string(mode), modeConfigJSON, id)
	if err != nil {
		return fmt.Errorf("update run mode %d: %w", id, err)
	}
	return nil
}

// UpdateRunURL stores the url and channel_id after WAITING_URL gate.
func (r *Repo) UpdateRunURL(ctx context.Context, id int64, url string, channelID *int64) error {
	var chID interface{} = nil
	if channelID != nil {
		chID = *channelID
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET url = ?, channel_id = ? WHERE id = ?`,
		url, chID, id)
	if err != nil {
		return fmt.Errorf("update run url %d: %w", id, err)
	}
	return nil
}

// UpdateRunPaths stores intermediate artifact paths.
func (r *Repo) UpdateRunPaths(ctx context.Context, id int64, videoPath, transcriptPath string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET video_path = ?, transcript_path = ? WHERE id = ?`,
		videoPath, transcriptPath, id)
	if err != nil {
		return fmt.Errorf("update run paths %d: %w", id, err)
	}
	return nil
}

// FinishRun sets finished_at, state=DONE/ERROR/CANCELLED and error message.
func (r *Repo) FinishRun(ctx context.Context, id int64, state RunState, errMsg string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET state = ?, error = ?, finished_at = ?, active_phase = '' WHERE id = ?`,
		string(state), errMsg, now, id)
	if err != nil {
		return fmt.Errorf("finish run %d: %w", id, err)
	}
	return nil
}

// DeleteRun removes a run and cascades to highlights and clips.
func (r *Repo) DeleteRun(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM pipeline_runs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete run %d: %w", id, err)
	}
	return nil
}

func scanRun(row *sql.Row) (*Run, error) {
	var p Run
	var channelID sql.NullInt64
	var finishedAt sql.NullInt64
	var mode, state string
	if err := row.Scan(
		&p.ID, &channelID, &p.URL, &mode, &state,
		&p.ActivePhase, &p.Error, &p.ModeConfigJSON,
		&p.VideoPath, &p.TranscriptPath, &p.StartedAt, &finishedAt,
	); err != nil {
		return nil, fmt.Errorf("scan run: %w", err)
	}
	p.Mode = WorkflowMode(mode)
	p.State = RunState(state)
	if channelID.Valid {
		v := channelID.Int64
		p.ChannelID = &v
	}
	if finishedAt.Valid {
		v := finishedAt.Int64
		p.FinishedAt = &v
	}
	return &p, nil
}

func scanRunRow(rows *sql.Rows) (*Run, error) {
	var p Run
	var channelID sql.NullInt64
	var finishedAt sql.NullInt64
	var mode, state string
	if err := rows.Scan(
		&p.ID, &channelID, &p.URL, &mode, &state,
		&p.ActivePhase, &p.Error, &p.ModeConfigJSON,
		&p.VideoPath, &p.TranscriptPath, &p.StartedAt, &finishedAt,
	); err != nil {
		return nil, fmt.Errorf("scan run row: %w", err)
	}
	p.Mode = WorkflowMode(mode)
	p.State = RunState(state)
	if channelID.Valid {
		v := channelID.Int64
		p.ChannelID = &v
	}
	if finishedAt.Valid {
		v := finishedAt.Int64
		p.FinishedAt = &v
	}
	return &p, nil
}

// ── pipeline_highlights ───────────────────────────────────────────────────────

const highlightCols = `id, run_id, start_sec, end_sec, adj_start_sec, adj_end_sec, score, text, reason, is_selected, created_at`

// InsertHighlights bulk-inserts highlights for a run.
func (r *Repo) InsertHighlights(ctx context.Context, runID int64, highlights []Highlight) error {
	if len(highlights) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	placeholders := make([]string, len(highlights))
	args := make([]interface{}, 0, len(highlights)*10)
	for i, h := range highlights {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(args, runID, h.StartSec, h.EndSec, h.AdjStartSec, h.AdjEndSec, h.Score, h.Text, h.Reason, boolToInt(h.IsSelected), now)
	}
	q := "INSERT INTO pipeline_highlights (run_id, start_sec, end_sec, adj_start_sec, adj_end_sec, score, text, reason, is_selected, created_at) VALUES " + strings.Join(placeholders, ",")
	_, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("insert highlights: %w", err)
	}
	return nil
}

// GetHighlights returns all highlights for a run.
func (r *Repo) GetHighlights(ctx context.Context, runID int64) ([]Highlight, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+highlightCols+" FROM pipeline_highlights WHERE run_id = ? ORDER BY start_sec", runID)
	if err != nil {
		return nil, fmt.Errorf("list highlights: %w", err)
	}
	defer rows.Close()
	var out []Highlight
	for rows.Next() {
		var h Highlight
		var isSelected int
		if err := rows.Scan(&h.ID, &h.RunID, &h.StartSec, &h.EndSec, &h.AdjStartSec, &h.AdjEndSec, &h.Score, &h.Text, &h.Reason, &isSelected, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan highlight: %w", err)
		}
		h.IsSelected = isSelected != 0
		out = append(out, h)
	}
	return out, rows.Err()
}

// UpdateHighlight applies user edits to a single highlight.
func (r *Repo) UpdateHighlight(ctx context.Context, id int64, adjStart, adjEnd float64, isSelected bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_highlights SET adj_start_sec = ?, adj_end_sec = ?, is_selected = ? WHERE id = ?`,
		adjStart, adjEnd, boolToInt(isSelected), id)
	if err != nil {
		return fmt.Errorf("update highlight %d: %w", id, err)
	}
	return nil
}

// ── pipeline_clips ────────────────────────────────────────────────────────────

const clipCols = `id, run_id, highlight_id, file_path, thumbnail_path, title, description, tags, thumbnail_style, is_selected, start_sec, end_sec, duration_sec, upload_status, youtube_id, created_at`

// InsertClip inserts a generated clip.
func (r *Repo) InsertClip(ctx context.Context, c *Clip) (int64, error) {
	now := time.Now().UnixMilli()
	var hlID interface{} = nil
	if c.HighlightID != nil {
		hlID = *c.HighlightID
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pipeline_clips (run_id, highlight_id, file_path, thumbnail_path, title, description, tags, thumbnail_style, is_selected, start_sec, end_sec, duration_sec, upload_status, youtube_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.RunID, hlID, c.FilePath, c.ThumbnailPath, c.Title, c.Description, c.Tags,
		c.ThumbnailStyle, boolToInt(c.IsSelected), c.StartSec, c.EndSec, c.DurationSec,
		c.UploadStatus, c.YoutubeID, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert clip: %w", err)
	}
	return res.LastInsertId()
}

// GetClips returns all clips for a run.
func (r *Repo) GetClips(ctx context.Context, runID int64) ([]Clip, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+clipCols+" FROM pipeline_clips WHERE run_id = ? ORDER BY id", runID)
	if err != nil {
		return nil, fmt.Errorf("list clips: %w", err)
	}
	defer rows.Close()
	var out []Clip
	for rows.Next() {
		c, err := scanClipRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// UpdateClipMetadata applies metadata edits from the review-metadata gate.
func (r *Repo) UpdateClipMetadata(ctx context.Context, id int64, title, description, tags string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_clips SET title = ?, description = ?, tags = ? WHERE id = ?`,
		title, description, tags, id)
	if err != nil {
		return fmt.Errorf("update clip metadata %d: %w", id, err)
	}
	return nil
}

// UpdateClipSelected sets is_selected for a clip.
func (r *Repo) UpdateClipSelected(ctx context.Context, id int64, selected bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_clips SET is_selected = ? WHERE id = ?`,
		boolToInt(selected), id)
	if err != nil {
		return fmt.Errorf("update clip selected %d: %w", id, err)
	}
	return nil
}

// UpdateClipThumbnailStyle updates the thumbnail style for a clip.
func (r *Repo) UpdateClipThumbnailStyle(ctx context.Context, id int64, style string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_clips SET thumbnail_style = ? WHERE id = ?`,
		style, id)
	if err != nil {
		return fmt.Errorf("update clip thumbnail style %d: %w", id, err)
	}
	return nil
}

// UpdateClipUpload stores upload result.
func (r *Repo) UpdateClipUpload(ctx context.Context, id int64, status, youtubeID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pipeline_clips SET upload_status = ?, youtube_id = ? WHERE id = ?`,
		status, youtubeID, id)
	if err != nil {
		return fmt.Errorf("update clip upload %d: %w", id, err)
	}
	return nil
}

func scanClipRow(rows *sql.Rows) (*Clip, error) {
	var c Clip
	var hlID sql.NullInt64
	var isSelected int
	if err := rows.Scan(
		&c.ID, &c.RunID, &hlID, &c.FilePath, &c.ThumbnailPath,
		&c.Title, &c.Description, &c.Tags, &c.ThumbnailStyle,
		&isSelected, &c.StartSec, &c.EndSec, &c.DurationSec,
		&c.UploadStatus, &c.YoutubeID, &c.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan clip row: %w", err)
	}
	c.IsSelected = isSelected != 0
	if hlID.Valid {
		v := hlID.Int64
		c.HighlightID = &v
	}
	return &c, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
