package worker

import (
	"context"
	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/queue"
	"fmt"
	"time"
)

// ImageWorker reads image jobs from the queue and processes them in the background
type ImageWorker struct {
	queue  *queue.ImageQueue
	logger *logger.Logger
}

// NewImageWorker creates a new ImageWorker
func NewImageWorker(q *queue.ImageQueue, log *logger.Logger) *ImageWorker {
	return &ImageWorker{queue: q, logger: log}
}

// Run starts the worker loop. Call this in a goroutine.
// It exits cleanly when ctx is cancelled (e.g. on server shutdown).
func (w *ImageWorker) Run(ctx context.Context) {
	w.logger.Info("Image worker started — listening for upload jobs")
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Image worker shutting down")
			return
		default:
			// Block for up to 5 s waiting for a job
			job, err := w.queue.Pop(ctx, 5*time.Second)
			if err != nil {
				if ctx.Err() != nil {
					return // context cancelled — normal shutdown
				}
				w.logger.Error("Image worker: queue pop error", err)
				time.Sleep(time.Second) // back off before retrying
				continue
			}
			if job == nil {
				continue // timeout — no job yet, loop again
			}
			w.process(job)
		}
	}
}

// process handles a single ImageJob.
// Extend this to add resizing, compression, virus scanning, CDN upload, etc.
func (w *ImageWorker) process(job *queue.ImageJob) {
	w.logger.Info(fmt.Sprintf(
		"[image-worker] processing job=%s file=%s user=%d size=%d mime=%s queued=%s",
		job.ID, job.FileName, job.UserID, job.SizeBytes, job.MimeType,
		job.QueuedAt.Format(time.RFC3339),
	))

	// ── Processing steps (add real logic here) ──────────────────────────────
	// 1. Validate image dimensions / size limits
	// 2. Strip EXIF metadata (privacy)
	// 3. Generate thumbnail
	// 4. Move from temp dir to permanent storage / CDN
	// 5. Update DB with final URL if needed
	// ────────────────────────────────────────────────────────────────────────

	w.logger.Info(fmt.Sprintf("[image-worker] job=%s done", job.ID))
}
