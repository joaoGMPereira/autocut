package database

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
)

// insertTestRun inserts a minimal pipeline_run row and returns its id.
func insertTestRun(t *testing.T, db *sql.DB, state string) int64 {
	t.Helper()
	res, err := db.ExecContext(context.Background(),
		`INSERT INTO pipeline_runs (channel_id, url, mode, state, active_phase, error, mode_config_json, video_path, transcript_path, started_at)
         VALUES (NULL, 'https://youtube.com/test', 'ai', ?, '', '', '{}', '', '', 0)`, state)
	if err != nil {
		t.Fatalf("insert test run: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestUpdateIsSelectedBatch_SelectSubset(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	repo := NewPipelineClipRepo(db, slog.Default())
	ctx := context.Background()

	runID := insertTestRun(t, db, "WAITING_REVIEW_CLIPS")
	c1, _ := repo.Create(ctx, &PipelineClip{RunID: runID, IsSelected: true, UploadStatus: "pending"})
	c2, _ := repo.Create(ctx, &PipelineClip{RunID: runID, IsSelected: true, UploadStatus: "pending"})
	c3, _ := repo.Create(ctx, &PipelineClip{RunID: runID, IsSelected: true, UploadStatus: "pending"})

	if err := repo.UpdateIsSelectedBatch(ctx, runID, []int64{c1, c3}); err != nil {
		t.Fatalf("UpdateIsSelectedBatch: %v", err)
	}

	clips, _ := repo.ListByRun(ctx, runID)
	sel := map[int64]bool{}
	for _, c := range clips {
		sel[c.ID] = c.IsSelected
	}
	if !sel[c1] {
		t.Error("c1 should be selected")
	}
	if sel[c2] {
		t.Error("c2 should NOT be selected")
	}
	if !sel[c3] {
		t.Error("c3 should be selected")
	}
}

func TestUpdateIsSelectedBatch_EmptyDeselectsAll(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	repo := NewPipelineClipRepo(db, slog.Default())
	ctx := context.Background()

	runID := insertTestRun(t, db, "WAITING_REVIEW_CLIPS")
	repo.Create(ctx, &PipelineClip{RunID: runID, IsSelected: true, UploadStatus: "pending"})
	repo.Create(ctx, &PipelineClip{RunID: runID, IsSelected: true, UploadStatus: "pending"})

	if err := repo.UpdateIsSelectedBatch(ctx, runID, []int64{}); err != nil {
		t.Fatalf("empty batch: %v", err)
	}
	clips, _ := repo.ListByRun(ctx, runID)
	for _, c := range clips {
		if c.IsSelected {
			t.Errorf("clip %d should be deselected", c.ID)
		}
	}
}

func TestUpdateTitleAndText(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	repo := NewPipelineClipRepo(db, slog.Default())
	ctx := context.Background()

	runID := insertTestRun(t, db, "WAITING_REVIEW_CLIPS")
	clipID, _ := repo.Create(ctx, &PipelineClip{RunID: runID, Title: "old", ThumbnailText: "OLD", UploadStatus: "pending"})

	if err := repo.UpdateTitleAndText(ctx, clipID, "New Title", "NEW TEXT"); err != nil {
		t.Fatalf("UpdateTitleAndText: %v", err)
	}
	clips, _ := repo.ListByRun(ctx, runID)
	if clips[0].Title != "New Title" {
		t.Errorf("title: got %q", clips[0].Title)
	}
	if clips[0].ThumbnailText != "NEW TEXT" {
		t.Errorf("thumb_text: got %q", clips[0].ThumbnailText)
	}
}
