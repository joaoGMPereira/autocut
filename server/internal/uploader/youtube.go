package uploader

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// YouTubeUploader performs all YouTube Data API v3 operations.
// Kotlin ref: YouTubeUploader (thin coordinator) + all delegate files
type YouTubeUploader struct {
	svc *youtube.Service
	log *slog.Logger
}

// YouTubeUploaderInterface is satisfied by YouTubeUploader and by test mocks.
type YouTubeUploaderInterface interface {
	Upload(ctx context.Context, req UploadRequest) (<-chan UploadProgress, error)
	UploadThumbnail(videoID, thumbnailPath string) error
	AddToPlaylist(videoID, playlistID string) error
	Schedule(videoID string, publishAt time.Time) error
	SetMetadata(videoID string, meta VideoMetadata) error
	GetComments(videoID string) ([]Comment, error)
	PinComment(videoID, text string) error
}

// NewYouTubeUploader creates a YouTubeUploader backed by the given HTTP client.
// The client must carry valid OAuth2 credentials (e.g. cfg.Client or oauth2.NewClient).
// Kotlin ref: AuthenticationDelegate.buildYouTubeService
func NewYouTubeUploader(httpClient *http.Client) (*YouTubeUploader, error) {
	svc, err := youtube.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create youtube service: %w", err)
	}
	return &YouTubeUploader{
		svc: svc,
		log: slog.With("component", "uploader.youtube"),
	}, nil
}

// Upload initiates a resumable video upload and streams progress events on the returned channel.
// The channel is closed after a Done=true event (success or error).
// Kotlin ref: VideoUploadDelegate.executeUpload + UploadProgressDelegate
func (u *YouTubeUploader) Upload(ctx context.Context, req UploadRequest) (<-chan UploadProgress, error) {
	f, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("open video file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat video file: %w", err)
	}
	totalBytes := info.Size()

	// Build video resource — snippet + status.
	vid := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:                req.Title,
			Description:          req.Description,
			Tags:                 req.Tags,
			CategoryId:           req.CategoryID,
			DefaultLanguage:      "pt-BR",
			DefaultAudioLanguage: "pt-BR",
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus:           privacyStatus(req.Privacy, req.ScheduleAt),
			SelfDeclaredMadeForKids: false,
		},
	}
	if req.ScheduleAt != nil {
		vid.Status.PublishAt = req.ScheduleAt.UTC().Format(time.RFC3339)
	}

	ch := make(chan UploadProgress, 32)

	go func() {
		defer f.Close()
		defer close(ch)

		// progressReader sends incremental events on ch while the upload reads bytes.
		pr := &progressReader{
			r:     f,
			total: totalBytes,
			ch:    ch,
			ctx:   ctx,
		}

		call := u.svc.Videos.Insert([]string{"snippet", "status"}, vid).Media(pr)

		returned, err := call.Do()
		if err != nil {
			u.log.Error("upload failed", "err", err, "title", req.Title)
			ch <- UploadProgress{
				TotalBytes: totalBytes,
				Done:       true,
				Err:        fmt.Errorf("upload: %w", err),
			}
			return
		}

		u.log.Info("upload complete", "videoID", returned.Id, "title", req.Title)
		ch <- UploadProgress{
			BytesSent:  totalBytes,
			TotalBytes: totalBytes,
			Percent:    100,
			VideoID:    returned.Id,
			Done:       true,
		}
	}()

	return ch, nil
}

// UploadThumbnail sets the custom thumbnail for videoID.
// Kotlin ref: ThumbnailUploadDelegate.uploadThumbnail
func (u *YouTubeUploader) UploadThumbnail(videoID, thumbnailPath string) error {
	f, err := os.Open(thumbnailPath)
	if err != nil {
		return fmt.Errorf("open thumbnail: %w", err)
	}
	defer f.Close()

	mime := thumbnailMIME(thumbnailPath)
	_, err = u.svc.Thumbnails.Set(videoID).Media(f, googleapi.ContentType(mime)).Do()
	if err != nil {
		return fmt.Errorf("set thumbnail for video %q: %w", videoID, err)
	}

	u.log.Info("thumbnail uploaded", "videoID", videoID, "path", thumbnailPath)
	return nil
}

// AddToPlaylist adds videoID to playlistID.
// Kotlin ref: PlaylistManagementDelegate.addToPlaylist
func (u *YouTubeUploader) AddToPlaylist(videoID, playlistID string) error {
	item := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistID,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoID,
			},
		},
	}
	_, err := u.svc.PlaylistItems.Insert([]string{"snippet"}, item).Do()
	if err != nil {
		return fmt.Errorf("add video %q to playlist %q: %w", videoID, playlistID, err)
	}
	u.log.Info("video added to playlist", "videoID", videoID, "playlistID", playlistID)
	return nil
}

// Schedule sets the publishAt time on a video, making it scheduled-private.
// Kotlin ref: SchedulingDelegate — updates status.publishAt
func (u *YouTubeUploader) Schedule(videoID string, publishAt time.Time) error {
	update := &youtube.Video{
		Id: videoID,
		Status: &youtube.VideoStatus{
			PrivacyStatus: "private",
			PublishAt:     publishAt.UTC().Format(time.RFC3339),
		},
	}
	_, err := u.svc.Videos.Update([]string{"status"}, update).Do()
	if err != nil {
		return fmt.Errorf("schedule video %q at %s: %w", videoID, publishAt.Format(time.RFC3339), err)
	}
	u.log.Info("video scheduled", "videoID", videoID, "publishAt", publishAt)
	return nil
}

// SetMetadata updates a video's snippet and optionally its privacy status.
// It fetches the existing snippet first to preserve fields not being changed.
// Kotlin ref: VideoManagementDelegate.updateVideo
func (u *YouTubeUploader) SetMetadata(videoID string, meta VideoMetadata) error {
	// Fetch current snippet to avoid clobbering category, language, etc.
	listResp, err := u.svc.Videos.List([]string{"snippet", "status"}).Id(videoID).Do()
	if err != nil {
		return fmt.Errorf("fetch video %q before update: %w", videoID, err)
	}
	if len(listResp.Items) == 0 {
		return fmt.Errorf("video %q not found", videoID)
	}

	snip := listResp.Items[0].Snippet
	snip.Title = meta.Title
	snip.Description = meta.Description
	snip.Tags = meta.Tags
	if meta.CategoryID != "" {
		snip.CategoryId = meta.CategoryID
	}

	update := &youtube.Video{
		Id:      videoID,
		Snippet: snip,
	}
	parts := []string{"snippet"}

	if meta.Privacy != "" {
		update.Status = &youtube.VideoStatus{PrivacyStatus: meta.Privacy}
		parts = append(parts, "status")
	}

	_, err = u.svc.Videos.Update(parts, update).Do()
	if err != nil {
		return fmt.Errorf("update metadata for video %q: %w", videoID, err)
	}

	u.log.Info("metadata updated", "videoID", videoID, "title", meta.Title)
	return nil
}

// GetComments returns the top-level comment threads for a video (up to 100).
// Kotlin ref: CommentManagementDelegate.hasComment / postComment
func (u *YouTubeUploader) GetComments(videoID string) ([]Comment, error) {
	resp, err := u.svc.CommentThreads.List([]string{"snippet"}).VideoId(videoID).MaxResults(100).Do()
	if err != nil {
		return nil, fmt.Errorf("list comments for video %q: %w", videoID, err)
	}

	comments := make([]Comment, 0, len(resp.Items))
	for _, thread := range resp.Items {
		snip := thread.Snippet.TopLevelComment.Snippet
		t, _ := time.Parse(time.RFC3339, snip.PublishedAt)
		comments = append(comments, Comment{
			ID:          thread.Id,
			AuthorName:  snip.AuthorDisplayName,
			Text:        snip.TextDisplay,
			PublishedAt: t,
			LikeCount:   snip.LikeCount,
		})
	}

	u.log.Info("comments retrieved", "videoID", videoID, "count", len(comments))
	return comments, nil
}

// PinComment posts a comment as the authenticated channel owner.
// NOTE: YouTube Data API v3 does not support a native "pin" endpoint.
// The comment is posted; manual pinning via YouTube Studio is required.
// Kotlin ref: CommentManagementDelegate.postComment + setPinnedComment note
func (u *YouTubeUploader) PinComment(videoID, text string) error {
	thread := &youtube.CommentThread{
		Snippet: &youtube.CommentThreadSnippet{
			VideoId: videoID,
			TopLevelComment: &youtube.Comment{
				Snippet: &youtube.CommentSnippet{
					TextOriginal: text,
				},
			},
		},
	}
	_, err := u.svc.CommentThreads.Insert([]string{"snippet"}, thread).Do()
	if err != nil {
		return fmt.Errorf("post comment on video %q: %w", videoID, err)
	}
	u.log.Info("comment posted (pin manually in Studio)", "videoID", videoID)
	return nil
}

// --- helpers ---------------------------------------------------------------

// progressReader wraps an io.Reader and sends UploadProgress events to ch.
// It never blocks the upload goroutine — slow consumers simply miss intermediate events.
type progressReader struct {
	r     io.Reader
	total int64
	sent  int64
	ch    chan<- UploadProgress
	ctx   context.Context
}

func (p *progressReader) Read(buf []byte) (int, error) {
	select {
	case <-p.ctx.Done():
		return 0, p.ctx.Err()
	default:
	}

	n, err := p.r.Read(buf)
	if n > 0 {
		p.sent += int64(n)
		pct := float64(p.sent) / float64(p.total) * 100
		select {
		case p.ch <- UploadProgress{
			BytesSent:  p.sent,
			TotalBytes: p.total,
			Percent:    pct,
		}:
		default: // non-blocking: skip if channel is full
		}
	}
	return n, err
}

// privacyStatus derives the correct YouTube privacy string.
// Scheduled videos must be uploaded as "private" — YouTube publishes at publishAt.
func privacyStatus(privacy string, scheduleAt *time.Time) string {
	if scheduleAt != nil {
		return "private"
	}
	switch privacy {
	case "public", "unlisted", "private":
		return privacy
	default:
		return "private"
	}
}

// thumbnailMIME maps a file extension to its MIME type.
// Kotlin ref: ThumbnailUploadDelegate.detectMimeType
func thumbnailMIME(path string) string {
	switch filepath.Ext(path) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
