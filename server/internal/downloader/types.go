package downloader

import "time"

// VideoInfo holds metadata and file information for a downloaded video.
// Kotlin ref: VideoInfo data class in VideoInfoDelegate.kt + Video model.
type VideoInfo struct {
	// VideoID is the platform-specific identifier (e.g. YouTube video ID).
	VideoID string

	// Title is the human-readable video title.
	Title string

	// Description is the full video description.
	Description string

	// ThumbnailURL is the remote URL of the video thumbnail.
	ThumbnailURL string

	// FilePath is the local path to the downloaded video file.
	FilePath string

	// Duration is the length of the video.
	Duration time.Duration
}
