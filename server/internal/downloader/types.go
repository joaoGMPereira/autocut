package downloader

import "time"

// VideoInfo is the primary metadata record for a downloaded or inspected video.
type VideoInfo struct {
	// VideoID is the YouTube video identifier (e.g. "dQw4w9WgXcQ").
	VideoID string

	// Title is the human-readable video title.
	Title string

	// Description is the full video description text.
	Description string

	// ThumbnailURL is the highest-resolution thumbnail URL provided by yt-dlp.
	ThumbnailURL string

	// ChannelName is the uploader/channel name from yt-dlp.
	ChannelName string

	// Duration is the video length.
	Duration time.Duration

	// FilePath is the absolute path to the downloaded video file on disk.
	// Populated by DownloadWithOptions; empty when returned by ExtractMetadata alone.
	FilePath string
}

// ytDlpJSON mirrors the yt-dlp --dump-json output fields the downloader cares about.
// It is unexported; callers always receive VideoInfo.
type ytDlpJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Duration    float64 `json:"duration"`
	Thumbnail   string  `json:"thumbnail"`
	Uploader    string  `json:"uploader"`
	Filename    string  `json:"_filename"`
}

// DownloadResult is kept for backward compatibility; VideoInfo.FilePath is preferred.
type DownloadResult struct {
	FilePath string
	VideoID  string
}

// DownloadStrategy describes a single yt-dlp invocation profile.
type DownloadStrategy struct {
	// Name is a human-readable label used in logs.
	Name string

	// ExtraArgs are appended to the base yt-dlp command.
	ExtraArgs []string

	// Format is the yt-dlp --format selector string.
	Format string

	// AllowCookies controls whether --cookies-from-browser is passed.
	AllowCookies bool
}
