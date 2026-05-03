package queue

import (
	"context"
	"encoding/json"
	"femProjectSqlc/internal/cache"
	"time"
)

// ImageQueueKey is the list key used as the image processing queue
const ImageQueueKey = "image:upload:queue"

// ImageJob represents a queued image processing task
type ImageJob struct {
	ID        string    `json:"id"`
	FilePath  string    `json:"file_path"`
	FileName  string    `json:"file_name"`
	UserID    int64     `json:"user_id"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	QueuedAt  time.Time `json:"queued_at"`
}

// ImageQueue wraps the cache for image job queueing
type ImageQueue struct {
	cache *cache.InMemoryCache
}

// NewImageQueue creates a new ImageQueue backed by the given InMemoryCache
func NewImageQueue(c *cache.InMemoryCache) *ImageQueue {
	return &ImageQueue{cache: c}
}

// Push enqueues an ImageJob onto the queue list
func (q *ImageQueue) Push(ctx context.Context, job ImageJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.cache.LPush(ctx, ImageQueueKey, data)
}

// Pop blocks for up to timeout waiting for a job.
// Returns (nil, nil) when the timeout expires with no job available.
func (q *ImageQueue) Pop(ctx context.Context, timeout time.Duration) (*ImageJob, error) {
	val, err := q.cache.BRPop(ctx, timeout, ImageQueueKey)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil
	}
	var job ImageJob
	if err := json.Unmarshal([]byte(val), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// Depth returns the number of pending jobs in the queue
func (q *ImageQueue) Depth(ctx context.Context) (int64, error) {
	return q.cache.QueueLength(ctx, ImageQueueKey)
}
