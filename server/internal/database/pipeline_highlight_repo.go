package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// PipelineHighlightRepo provides read access for pipeline_highlights (v9 schema).
type PipelineHighlightRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewPipelineHighlightRepo(db *sql.DB, log *slog.Logger) *PipelineHighlightRepo {
	return &PipelineHighlightRepo{db: db, log: log.With("repo", "pipeline_highlight")}
}

const pipelineHighlightCols = `id, run_id, start_sec, end_sec, adj_start_sec, adj_end_sec, score, text, reason, is_selected, created_at`

// ListSelectedByRun returns all selected highlights for a given run, ordered by start_sec.
func (r *PipelineHighlightRepo) ListSelectedByRun(ctx context.Context, runID int64) ([]PipelineHighlight, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+pipelineHighlightCols+" FROM pipeline_highlights WHERE run_id = ? AND is_selected = 1 ORDER BY start_sec ASC",
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list selected pipeline_highlights for run %d: %w", runID, err)
	}
	defer rows.Close()

	var result []PipelineHighlight
	for rows.Next() {
		h, err := scanPipelineHighlightRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *h)
	}
	return result, rows.Err()
}

// DeleteByRun deletes all highlights for a run. Intended to be called within ReplaceForRun transaction.
func (r *PipelineHighlightRepo) DeleteByRun(ctx context.Context, runID int64) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM pipeline_highlights WHERE run_id = ?", runID)
	if err != nil {
		return fmt.Errorf("delete pipeline_highlights for run %d: %w", runID, err)
	}
	return nil
}

// BulkInsert inserts multiple highlights for the same run.
func (r *PipelineHighlightRepo) BulkInsert(ctx context.Context, highlights []PipelineHighlight) error {
	now := time.Now().UnixMilli()
	for _, h := range highlights {
		createdAt := h.CreatedAt
		if createdAt == 0 {
			createdAt = now
		}
		_, err := r.db.ExecContext(ctx,
			`INSERT INTO pipeline_highlights
			 (run_id, start_sec, end_sec, adj_start_sec, adj_end_sec, score, text, reason, is_selected, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			h.RunID, h.StartSec, h.EndSec, h.AdjStartSec, h.AdjEndSec, h.Score, h.Text, h.Reason, h.IsSelected, createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert pipeline_highlight for run %d: %w", h.RunID, err)
		}
	}
	return nil
}

// ReplaceForRun atomically clears all prior highlights for runID and inserts new ones.
// Uses a single transaction so no partial state is visible to concurrent readers.
func (r *PipelineHighlightRepo) ReplaceForRun(ctx context.Context, runID int64, highlights []PipelineHighlight) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for replace highlights run %d: %w", runID, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, "DELETE FROM pipeline_highlights WHERE run_id = ?", runID); err != nil {
		return fmt.Errorf("delete highlights in tx run %d: %w", runID, err)
	}

	now := time.Now().UnixMilli()
	for _, h := range highlights {
		createdAt := h.CreatedAt
		if createdAt == 0 {
			createdAt = now
		}
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO pipeline_highlights
			 (run_id, start_sec, end_sec, adj_start_sec, adj_end_sec, score, text, reason, is_selected, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			h.RunID, h.StartSec, h.EndSec, h.AdjStartSec, h.AdjEndSec, h.Score, h.Text, h.Reason, h.IsSelected, createdAt,
		); err != nil {
			return fmt.Errorf("insert highlight in tx run %d: %w", runID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit replace highlights run %d: %w", runID, err)
	}
	r.log.Info("replaced highlights", "run_id", runID, "count", len(highlights))
	return nil
}

func scanPipelineHighlightRows(rows *sql.Rows) (*PipelineHighlight, error) {
	var h PipelineHighlight
	err := rows.Scan(&h.ID, &h.RunID, &h.StartSec, &h.EndSec, &h.AdjStartSec, &h.AdjEndSec, &h.Score, &h.Text, &h.Reason, &h.IsSelected, &h.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan pipeline_highlight: %w", err)
	}
	return &h, nil
}
