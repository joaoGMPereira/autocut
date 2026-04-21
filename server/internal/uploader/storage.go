package uploader

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// QueueStorage manages the upload_queue directory on disk.
type QueueStorage struct {
	dataDir string
}

// NewQueueStorage creates a QueueStorage rooted at dataDir.
func NewQueueStorage(dataDir string) *QueueStorage {
	return &QueueStorage{dataDir: dataDir}
}

// SaveToQueue copies video + optional thumbnail to a new upload_queue subdirectory,
// writes metadata.json, and returns (queuedVideoPath, queuedThumbnailPath, error).
// If thumbnailPath is empty, the thumbnail step is skipped and queuedThumb is "".
func (q *QueueStorage) SaveToQueue(videoPath, thumbnailPath string, metadata VideoMetadata) (queuedVideo, queuedThumb string, err error) {
	ts := time.Now().UnixMilli()
	rnd := rand.Int63n(9000) + 1000 //nolint:gosec
	dirName := fmt.Sprintf("%d_%d", ts, rnd)
	queueDir := filepath.Join(q.dataDir, "upload_queue", dirName)
	if err := os.MkdirAll(queueDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create queue dir: %w", err)
	}

	// Copy video
	dstVideo := filepath.Join(queueDir, "video.mp4")
	if err := queueCopyFile(videoPath, dstVideo); err != nil {
		return "", "", fmt.Errorf("copy video: %w", err)
	}

	// Copy thumbnail (optional — non-fatal if source empty)
	dstThumb := ""
	if thumbnailPath != "" {
		candidate := filepath.Join(queueDir, "thumbnail.jpg")
		if cpErr := queueCopyFile(thumbnailPath, candidate); cpErr == nil {
			dstThumb = candidate
		}
	}

	// Write metadata.json
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(queueDir, "metadata.json"), metaBytes, 0o644); err != nil {
		return "", "", fmt.Errorf("write metadata.json: %w", err)
	}

	return dstVideo, dstThumb, nil
}

func queueCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return out.Close()
}
