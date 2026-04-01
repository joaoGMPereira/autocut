package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// PipelineRunRepo provides CRUD for pipeline_runs.
// Kotlin ref: PipelineRunsTable in Tables.kt
type PipelineRunRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewPipelineRunRepo(db *sql.DB, log *slog.Logger) *PipelineRunRepo {
	return &PipelineRunRepo{db: db, log: log.With("repo", "pipeline_run")}
}

const pipelineRunCols = `id, channel_id, url, mode, status, progress, current_step, error, step_outputs_json, mode_config_json, started_at, finished_at`

func (r *PipelineRunRepo) Create(ctx context.Context, p *PipelineRun) (int64, error) {
	now := time.Now().UnixMilli()
	stepOutputs := p.StepOutputsJSON
	if stepOutputs == "" {
		stepOutputs = "{}"
	}
	modeConfig := p.ModeConfigJSON
	if modeConfig == "" {
		modeConfig = "{}"
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pipeline_runs (channel_id, url, mode, status, progress, current_step, error, step_outputs_json, mode_config_json, started_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ChannelID, p.URL, p.Mode, p.Status, p.Progress, p.CurrentStep, p.Error, stepOutputs, modeConfig, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert pipeline_run: %w", err)
	}
	return res.LastInsertId()
}

func (r *PipelineRunRepo) GetByID(ctx context.Context, id int64) (*PipelineRun, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+pipelineRunCols+" FROM pipeline_runs WHERE id = ?", id)
	return scanPipelineRun(row)
}

// GetWithSteps returns a pipeline run together with its steps.
func (r *PipelineRunRepo) GetWithSteps(ctx context.Context, id int64) (*PipelineRun, []PipelineRunStep, error) {
	p, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, run_id, step, status, job_id, error, started_at, finished_at
		 FROM pipeline_run_steps WHERE run_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list steps for run %d: %w", id, err)
	}
	defer rows.Close()

	var steps []PipelineRunStep
	for rows.Next() {
		var s PipelineRunStep
		if err := rows.Scan(&s.ID, &s.RunID, &s.Step, &s.Status, &s.JobID, &s.Error, &s.StartedAt, &s.FinishedAt); err != nil {
			return nil, nil, fmt.Errorf("scan pipeline_run_step: %w", err)
		}
		steps = append(steps, s)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return p, steps, nil
}

func (r *PipelineRunRepo) ListByChannel(ctx context.Context, channelID int64) ([]PipelineRun, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+pipelineRunCols+" FROM pipeline_runs WHERE channel_id = ? ORDER BY started_at DESC", channelID)
	if err != nil {
		return nil, fmt.Errorf("list pipeline_runs by channel: %w", err)
	}
	defer rows.Close()

	var result []PipelineRun
	for rows.Next() {
		p, err := scanPipelineRunRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	return result, rows.Err()
}

// ListAll returns paginated pipeline runs with optional channel_id filter. Returns runs and total count.
func (r *PipelineRunRepo) ListAll(ctx context.Context, limit, offset int, channelID *int64) ([]PipelineRun, int, error) {
	var countQuery, dataQuery string
	var args []interface{}

	if channelID != nil {
		countQuery = "SELECT COUNT(*) FROM pipeline_runs WHERE channel_id = ?"
		dataQuery = "SELECT " + pipelineRunCols + " FROM pipeline_runs WHERE channel_id = ? ORDER BY started_at DESC LIMIT ? OFFSET ?"
		args = []interface{}{*channelID}
	} else {
		countQuery = "SELECT COUNT(*) FROM pipeline_runs"
		dataQuery = "SELECT " + pipelineRunCols + " FROM pipeline_runs ORDER BY started_at DESC LIMIT ? OFFSET ?"
	}

	var total int
	if channelID != nil {
		if err := r.db.QueryRowContext(ctx, countQuery, *channelID).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count pipeline_runs: %w", err)
		}
	} else {
		if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count pipeline_runs: %w", err)
		}
	}

	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, queryArgs...)
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
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

// UpdateProgress updates progress percentage and current step description.
func (r *PipelineRunRepo) UpdateProgress(ctx context.Context, id int64, progress int, step string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET progress = ?, current_step = ? WHERE id = ?",
		progress, step, id,
	)
	if err != nil {
		return fmt.Errorf("update pipeline_run progress %d: %w", id, err)
	}
	return nil
}

// UpdateMode updates the mode and mode_config_json for a pipeline run.
func (r *PipelineRunRepo) UpdateMode(ctx context.Context, id int64, mode string, modeConfigJSON string) error {
	if modeConfigJSON == "" {
		modeConfigJSON = "{}"
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET mode = ?, mode_config_json = ? WHERE id = ?",
		mode, modeConfigJSON, id,
	)
	if err != nil {
		return fmt.Errorf("update pipeline_run mode %d: %w", id, err)
	}
	return nil
}

// UpdateStepOutput updates the step_outputs_json field for a pipeline run.
func (r *PipelineRunRepo) UpdateStepOutput(ctx context.Context, id int64, stepOutputsJSON string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET step_outputs_json = ? WHERE id = ?",
		stepOutputsJSON, id,
	)
	if err != nil {
		return fmt.Errorf("update pipeline_run step_outputs %d: %w", id, err)
	}
	return nil
}

// Finish marks a pipeline run as completed or errored with final status.
func (r *PipelineRunRepo) Finish(ctx context.Context, id int64, status string, errMsg string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_runs SET status = ?, error = ?, finished_at = ? WHERE id = ?",
		status, errMsg, now, id,
	)
	if err != nil {
		return fmt.Errorf("finish pipeline_run %d: %w", id, err)
	}
	return nil
}

// Delete removes a pipeline run and its steps (via CASCADE).
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

// CreateStep inserts a new pipeline run step.
func (r *PipelineRunRepo) CreateStep(ctx context.Context, s *PipelineRunStep) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pipeline_run_steps (run_id, step, status, job_id, error, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.RunID, s.Step, s.Status, s.JobID, s.Error, s.StartedAt, s.FinishedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert pipeline_run_step: %w", err)
	}
	return res.LastInsertId()
}

// UpdateStep updates status, error, and finished_at for a pipeline run step.
func (r *PipelineRunRepo) UpdateStep(ctx context.Context, id int64, status string, errMsg string, finishedAt sql.NullInt64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE pipeline_run_steps SET status = ?, error = ?, finished_at = ? WHERE id = ?",
		status, errMsg, finishedAt, id,
	)
	if err != nil {
		return fmt.Errorf("update pipeline_run_step %d: %w", id, err)
	}
	return nil
}

func scanPipelineRun(row *sql.Row) (*PipelineRun, error) {
	var p PipelineRun
	err := row.Scan(&p.ID, &p.ChannelID, &p.URL, &p.Mode, &p.Status, &p.Progress, &p.CurrentStep, &p.Error, &p.StepOutputsJSON, &p.ModeConfigJSON, &p.StartedAt, &p.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPipelineRunRows(rows *sql.Rows) (*PipelineRun, error) {
	var p PipelineRun
	err := rows.Scan(&p.ID, &p.ChannelID, &p.URL, &p.Mode, &p.Status, &p.Progress, &p.CurrentStep, &p.Error, &p.StepOutputsJSON, &p.ModeConfigJSON, &p.StartedAt, &p.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
