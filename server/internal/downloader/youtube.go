package downloader

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ytDlpJSON mirrors the relevant fields in yt-dlp's --dump-json output.
// Kotlin ref: JsonParser.extractString / extractDouble in VideoInfoDelegate.kt.
type ytDlpJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Duration    float64 `json:"duration"` // seconds (float in yt-dlp)
	Thumbnail   string  `json:"thumbnail"`
	Filename    string  `json:"_filename"`
}

// YouTubeDownloader wraps yt-dlp for video download and metadata extraction.
// Kotlin ref: YouTubeDownloader + VideoDownloadDelegate + VideoInfoDelegate.
type YouTubeDownloader struct {
	binPath  string
	executor Executor
	retry    RetryConfig
	log      *slog.Logger
}

// NewYouTubeDownloader creates a YouTubeDownloader using the given yt-dlp binary path.
// Pass an empty string to use the default "yt-dlp" from PATH.
func NewYouTubeDownloader(binPath string) *YouTubeDownloader {
	if binPath == "" {
		binPath = "yt-dlp"
	}
	return &YouTubeDownloader{
		binPath:  binPath,
		executor: &DefaultExecutor{},
		retry:    DefaultRetryConfig,
		log:      slog.With("component", "downloader", "platform", "youtube"),
	}
}

// withExecutor returns a copy with a custom executor (used in tests).
func (d *YouTubeDownloader) withExecutor(ex Executor) *YouTubeDownloader {
	cp := *d
	cp.executor = ex
	return &cp
}

// ExtractMetadata fetches video metadata via yt-dlp --dump-json without downloading.
// Kotlin ref: VideoInfoDelegate.getDetailedVideoInfo()
func (d *YouTubeDownloader) ExtractMetadata(url string) (*VideoInfo, error) {
	d.log.Info("extracting metadata", "url", url)

	info, err := Retry(d.retry, func() (*VideoInfo, error) {
		out, err := d.executor.Run(d.binPath,
			"--dump-json",
			"--no-download",
			"--no-playlist",
			url,
		)
		if err != nil {
			return nil, fmt.Errorf("yt-dlp dump-json: %w", err)
		}
		return parseYtDlpJSON(out)
	})
	if err != nil {
		d.log.Error("metadata extraction failed", "err", err, "url", url)
		return nil, fmt.Errorf("extract metadata: %w", err)
	}

	d.log.Info("metadata extracted", "videoID", info.VideoID, "title", info.Title)
	return info, nil
}

// Download downloads a video to outDir and returns full VideoInfo.
// Output filename: <videoID>.%(ext)s → yt-dlp resolves final extension.
// Kotlin ref: VideoDownloadDelegate.download()
func (d *YouTubeDownloader) Download(url, outDir string) (*VideoInfo, error) {
	return d.DownloadWithOptions(url, outDir, "")
}

// qualityToFormat maps a quality label to the yt-dlp --format string.
func qualityToFormat(quality string) string {
	switch quality {
	case "720p":
		return "bestvideo[height<=720]+bestaudio/best[height<=720]"
	case "1080p":
		return "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	case "best":
		return "bestvideo+bestaudio/best"
	default:
		return "bestvideo[height<=1080]+bestaudio/best"
	}
}

// DownloadWithOptions downloads a video to outDir with the given quality
// preference and returns full VideoInfo.
func (d *YouTubeDownloader) DownloadWithOptions(url, outDir, quality string) (*VideoInfo, error) {
	d.log.Info("download started", "url", url, "outDir", outDir, "quality", quality)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create outDir: %w", err)
	}

	// Cache check: reuse existing file if already downloaded.
	if videoID := extractYouTubeVideoID(url); videoID != "" {
		if cached := findVideoFile(outDir, videoID); cached != "" {
			d.log.Info("download cache hit — reusing existing file", "videoID", videoID, "filePath", cached)
			meta, err := d.ExtractMetadata(url)
			if err != nil {
				d.log.Warn("metadata unavailable for cached file", "err", err, "videoID", videoID)
				meta = &VideoInfo{VideoID: videoID}
			}
			meta.FilePath = cached
			return meta, nil
		}
	}

	outputTemplate := filepath.Join(outDir, "%(id)s.%(ext)s")
	format := qualityToFormat(quality)

	info, err := Retry(d.retry, func() (*VideoInfo, error) {
		_, err := d.executor.Run(d.binPath,
			"--format", format,
			"--merge-output-format", "mp4",
			"--output", outputTemplate,
			"--no-playlist",
			"--newline",
			"--no-colors",
			url,
		)
		if err != nil {
			return nil, fmt.Errorf("yt-dlp download: %w", err)
		}
		return nil, nil
	})
	if err != nil {
		d.log.Error("download failed", "err", err, "url", url)
		return nil, fmt.Errorf("download: %w", err)
	}
	_ = info // Retry[*VideoInfo] — actual info comes from metadata call below

	// Fetch metadata after successful download.
	meta, err := d.ExtractMetadata(url)
	if err != nil {
		// Non-fatal: return partial info with empty metadata.
		d.log.Warn("metadata unavailable after download", "err", err, "url", url)
		meta = &VideoInfo{}
	}

	// Resolve the actual file path on disk.
	meta.FilePath = findVideoFile(outDir, meta.VideoID)

	d.log.Info("download complete",
		"videoID", meta.VideoID,
		"filePath", meta.FilePath,
	)
	return meta, nil
}

// DownloadThumbnail downloads the thumbnail for url and saves it to destPath.
// Kotlin ref: ThumbnailDownloadDelegate.downloadThumbnailFromYouTube()
func (d *YouTubeDownloader) DownloadThumbnail(url, destPath string) error {
	d.log.Info("thumbnail download started", "url", url, "dest", destPath)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create thumbnail dir: %w", err)
	}

	_, err := Retry(d.retry, func() (struct{}, error) {
		_, err := d.executor.Run(d.binPath,
			"--skip-download",
			"--write-thumbnail",
			"--convert-thumbnails", "jpg",
			"--output", destPath,
			url,
		)
		if err != nil {
			return struct{}{}, fmt.Errorf("yt-dlp thumbnail: %w", err)
		}
		return struct{}{}, nil
	})
	if err != nil {
		d.log.Error("thumbnail download failed", "err", err, "url", url)
		return fmt.Errorf("download thumbnail: %w", err)
	}

	d.log.Info("thumbnail downloaded", "dest", destPath)
	return nil
}

// parseYtDlpJSON decodes yt-dlp --dump-json output into VideoInfo.
func parseYtDlpJSON(data []byte) (*VideoInfo, error) {
	var j ytDlpJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("parse yt-dlp json: %w", err)
	}
	return &VideoInfo{
		VideoID:      j.ID,
		Title:        j.Title,
		Description:  j.Description,
		ThumbnailURL: j.Thumbnail,
		FilePath:     j.Filename,
		Duration:     time.Duration(j.Duration * float64(time.Second)),
	}, nil
}

// findVideoFile searches outDir for a video file matching videoID.
// Returns empty string if not found.
func findVideoFile(outDir, videoID string) string {
	for _, ext := range []string{".mp4", ".mkv", ".webm"} {
		candidate := filepath.Join(outDir, videoID+ext)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// extractYouTubeVideoID parses a YouTube URL and returns the video ID.
// Supports youtube.com (?v=...) and youtu.be (/<id>) URL forms.
// Returns "" on any error or if no ID is found.
func extractYouTubeVideoID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	switch u.Hostname() {
	case "www.youtube.com", "youtube.com", "m.youtube.com":
		return u.Query().Get("v")
	case "youtu.be":
		// Path is "/<id>"; trim the leading slash and take the first segment.
		p := strings.TrimPrefix(u.Path, "/")
		if idx := strings.IndexByte(p, '/'); idx != -1 {
			p = p[:idx]
		}
		return p
	}
	return ""
}

// httpGet is a package-level variable so tests can override the HTTP client.
var httpGet = http.Get
