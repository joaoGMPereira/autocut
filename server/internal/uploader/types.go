package uploader

import "time"

// UploadRequest holds everything needed to upload a video to YouTube.
// Kotlin ref: YouTubeVideoMetadata
type UploadRequest struct {
	FilePath    string
	Title       string
	Description string
	Tags        []string
	Privacy     string     // "public" | "unlisted" | "private"
	CategoryID  string     // numeric string, e.g. "22"
	PlaylistID  string     // optional; added after upload if non-empty
	ScheduleAt  *time.Time // nil = upload immediately
}

// UploadProgress is streamed on the channel returned by YouTubeUploader.Upload.
// Done=true marks the final event (either success or error).
// Kotlin ref: UploadProgressDelegate (state reported per chunk)
type UploadProgress struct {
	BytesSent  int64
	TotalBytes int64
	Percent    float64
	VideoID    string // populated once upload completes
	Done       bool
	Err        error
}

// VideoMetadata is used for bulk/partial updates via SetMetadata.
// Kotlin ref: YouTubeVideoMetadata (fields used in updateVideo)
type VideoMetadata struct {
	VideoID     string
	Title       string
	Description string
	Tags        []string
	CategoryID  string
	Privacy     string
}

// Comment represents a single top-level YouTube comment thread.
// Kotlin ref: CommentManagementDelegate.postComment / hasComment
type Comment struct {
	ID          string
	AuthorName  string
	Text        string
	PublishedAt time.Time
	LikeCount   int64
}
