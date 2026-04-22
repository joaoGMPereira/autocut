package uploader

import (
	"context"
	"log/slog"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// Worker polls the upload queue and triggers YouTube uploads when items are due.
type Worker struct {
	uploader   *YouTubeUploader
	uploadRepo *database.UploadRepo
	log        *slog.Logger
}

// NewWorker constructs an upload Worker.
func NewWorker(uploader *YouTubeUploader, uploadRepo *database.UploadRepo) *Worker {
	return &Worker{
		uploader:   uploader,
		uploadRepo: uploadRepo,
		log:        slog.With("component", "upload_worker"),
	}
}

// Trigger runs the upload loop once immediately (non-blocking, safe to call concurrently).
func (w *Worker) Trigger(ctx context.Context) {
	go w.runOnce(ctx)
}

// Start begins the polling loop. Blocks until ctx is cancelled.
// Run as a goroutine: go worker.Start(ctx).
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	w.log.Info("upload worker started", "interval", "60s")
	w.runOnce(ctx) // process any due items immediately on startup
	for {
		select {
		case <-ticker.C:
			w.runOnce(ctx)
		case <-ctx.Done():
			w.log.Info("upload worker stopped")
			return
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	items, err := w.uploadRepo.ListDueForUpload(ctx)
	if err != nil {
		w.log.Error("list due uploads", "err", err)
		return
	}
	if len(items) == 0 {
		return
	}
	w.log.Info("processing due uploads", "count", len(items))
	for _, item := range items {
		if ctx.Err() != nil {
			return
		}
		// Mark running to prevent double-dispatch on overlapping ticks
		if setErr := w.uploadRepo.UpdateStatus(ctx, item.ID, "running"); setErr != nil {
			w.log.Error("mark running failed", "id", item.ID, "err", setErr)
			continue
		}
		if uploadErr := w.uploader.Upload(ctx, item); uploadErr != nil {
			w.log.Error("upload failed", "id", item.ID, "err", uploadErr)
			_ = w.uploadRepo.MarkError(ctx, item.ID, uploadErr.Error())
		} else {
			w.log.Info("upload completed", "id", item.ID)
		}
	}
}
