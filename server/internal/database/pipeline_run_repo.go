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

const pipelineRunCols = `id, channel_id, url, mode, status, progress, current_step, error, started_at, finished_at`

func (r *PipelineRunRepo) Create(ctx context.Context, p *PipelineRun) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO pipeline_runs (channel_id, url, mode, status, progress, current_step, error, started_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ChannelID, p.URL, p.Mode, p.Status, p.Progress, p.CurrentStep, p.Error, now,
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

func scanPipelineRun(row *sql.Row) (*PipelineRun, error) {
	var p PipelineRun
	err := row.Scan(&p.ID, &p.ChannelID, &p.URL, &p.Mode, &p.Status, &p.Progress, &p.CurrentStep, &p.Error, &p.StartedAt, &p.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPipelineRunRows(rows *sql.Rows) (*PipelineRun, error) {
	var p PipelineRun
	err := rows.Scan(&p.ID, &p.ChannelID, &p.URL, &p.Mode, &p.Status, &p.Progress, &p.CurrentStep, &p.Error, &p.StartedAt, &p.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
