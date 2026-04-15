package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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

func scanPipelineHighlightRows(rows *sql.Rows) (*PipelineHighlight, error) {
	var h PipelineHighlight
	err := rows.Scan(&h.ID, &h.RunID, &h.StartSec, &h.EndSec, &h.AdjStartSec, &h.AdjEndSec, &h.Score, &h.Text, &h.Reason, &h.IsSelected, &h.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan pipeline_highlight: %w", err)
	}
	return &h, nil
}
