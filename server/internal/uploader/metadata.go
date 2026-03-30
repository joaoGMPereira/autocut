package uploader

import (
	"fmt"
	"log/slog"
	"strings"
)

// MetadataManager orchestrates bulk metadata updates and pinned-comment synchronisation.
// Kotlin ref: VideoManagementDelegate.updateVideo + CommentManagementDelegate (template logic)
type MetadataManager struct {
	uploader YouTubeUploaderInterface
	log      *slog.Logger
}

// NewMetadataManager creates a MetadataManager backed by uploader.
func NewMetadataManager(uploader YouTubeUploaderInterface) *MetadataManager {
	return &MetadataManager{
		uploader: uploader,
		log:      slog.With("component", "uploader.metadata"),
	}
}

// BulkUpdate applies SetMetadata sequentially for each entry in videos.
// On first error it stops and returns it, preserving the index for the caller.
// Kotlin ref: YouTubeUploader.uploadBatch (sequential strategy)
func (m *MetadataManager) BulkUpdate(videos []VideoMetadata) error {
	for i, v := range videos {
		if err := m.uploader.SetMetadata(v.VideoID, v); err != nil {
			return fmt.Errorf("bulk update [%d/%d] videoID=%q: %w", i+1, len(videos), v.VideoID, err)
		}
		m.log.Info("bulk update progress", "index", i+1, "total", len(videos), "videoID", v.VideoID)
	}
	return nil
}

// SyncPinnedComment resolves template variable placeholders and posts the result as a comment.
// Placeholders use the form {key}; unmatched keys are left as-is.
// Kotlin ref: CommentManagementDelegate.syncCommentsDeep — template substitution step
func (m *MetadataManager) SyncPinnedComment(videoID, template string, vars map[string]string) error {
	text := resolvePlaceholders(template, vars)
	if err := m.uploader.PinComment(videoID, text); err != nil {
		return fmt.Errorf("sync pinned comment for video %q: %w", videoID, err)
	}
	m.log.Info("pinned comment synced", "videoID", videoID)
	return nil
}

// resolvePlaceholders replaces every {key} in tmpl with vars[key].
// Unknown keys are left unchanged so the caller can detect them.
func resolvePlaceholders(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}
