package downloader

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// YouTubeDownloader downloads and inspects YouTube videos using yt-dlp.
type YouTubeDownloader struct {
	binary   string
	executor Executor
	retry    RetryConfig
}

// NewYouTubeDownloader creates a downloader backed by the given binary name.
// If binary is empty, it defaults to "yt-dlp".
func NewYouTubeDownloader(binary string) *YouTubeDownloader {
	if binary == "" {
		binary = "yt-dlp"
	}
	return &YouTubeDownloader{
		binary:   binary,
		executor: OSExecutor{},
		retry:    RetryConfig{MaxAttempts: 3, InitialDelay: time.Second},
	}
}

// withExecutor replaces the executor. Intended for tests only.
func (d *YouTubeDownloader) withExecutor(e Executor) *YouTubeDownloader {
	d.executor = e
	return d
}

// ExtractMetadata calls yt-dlp with --dump-json --no-download --no-playlist
// and unmarshals the JSON output into a VideoInfo.
// Retries according to d.retry on executor errors.
func (d *YouTubeDownloader) ExtractMetadata(rawURL string) (VideoInfo, error) {
	return Retry(d.retry, func() (VideoInfo, error) {
		out, err := d.executor.Run(d.binary, "--dump-json", "--no-download", "--no-playlist", rawURL)
		if err != nil {
			return VideoInfo{}, err
		}
		var parsed ytDlpJSON
		if err := json.Unmarshal(out, &parsed); err != nil {
			return VideoInfo{}, err
		}
		return VideoInfo{
			VideoID:      parsed.ID,
			Title:        parsed.Title,
			Description:  parsed.Description,
			ThumbnailURL: parsed.Thumbnail,
			Duration:     time.Duration(parsed.Duration * float64(time.Second)),
		}, nil
	})
}

// extractVideoID parses the YouTube video ID from a URL.
// Supports standard watch URLs (v= param) and short youtu.be URLs.
func extractVideoID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Host == "youtu.be" {
		return strings.TrimPrefix(u.Path, "/")
	}
	return u.Query().Get("v")
}

// detectBrowser returns the name of the first installed browser whose profile
// directory exists, or "" if none is found.
func detectBrowser() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	type candidate struct {
		name string
		path string
	}
	var candidates []candidate
	switch runtime.GOOS {
	case "darwin":
		candidates = []candidate{
			{"chrome", filepath.Join(home, "Library", "Application Support", "Google", "Chrome")},
			{"firefox", filepath.Join(home, "Library", "Application Support", "Firefox")},
			{"brave", filepath.Join(home, "Library", "Application Support", "BraveSoftware", "Brave-Browser")},
			{"edge", filepath.Join(home, "Library", "Application Support", "Microsoft Edge")},
			{"chromium", filepath.Join(home, "Library", "Application Support", "Chromium")},
		}
	case "linux":
		candidates = []candidate{
			{"chrome", filepath.Join(home, ".config", "google-chrome")},
			{"firefox", filepath.Join(home, ".mozilla", "firefox")},
			{"brave", filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser")},
			{"chromium", filepath.Join(home, ".config", "chromium")},
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c.path); err == nil {
			return c.name
		}
	}
	return ""
}

// downloadStrategies returns the ordered list of fallback strategies for
// a given quality ceiling (e.g. 1080).
func downloadStrategies(qualityInt int) []DownloadStrategy {
	qual := fmt.Sprintf("bestvideo[height<=%d]+bestaudio/best", qualityInt)
	return []DownloadStrategy{
		{
			Name:         "web_client_with_cookies",
			Format:       qual,
			AllowCookies: true,
		},
		{
			Name:         "web_client_no_cookies",
			Format:       qual,
			AllowCookies: false,
		},
		{
			Name:      "android_client",
			ExtraArgs: []string{"--extractor-args", "youtube:player_client=android"},
			Format:    qual,
		},
		{
			Name:      "ios_client",
			ExtraArgs: []string{"--extractor-args", "youtube:player_client=ios"},
			Format:    qual,
		},
		{
			Name:      "tv_embedded_client",
			ExtraArgs: []string{"--extractor-args", "youtube:player_client=tv_embedded"},
			Format:    qual,
		},
		{
			Name:      "progressive_720p",
			ExtraArgs: []string{"--extractor-args", "youtube:player_client=android"},
			Format:    "best[height<=720][ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		},
		{
			Name:   "best_available",
			Format: "best",
		},
	}
}

// isChallenge reports whether output looks like a YouTube bot-challenge or 403 error.
func isChallenge(output []byte) bool {
	s := strings.ToLower(string(output))
	for _, kw := range []string{"n challenge", "403", "botguard", "droidguard", "sign in to confirm", "nsig"} {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// buildDownloadArgs constructs the yt-dlp argument list for a download strategy.
func buildDownloadArgs(strategy DownloadStrategy, outputTemplate, videoURL string) []string {
	args := []string{
		"--format", strategy.Format,
		"--merge-output-format", "mp4",
		"--output", outputTemplate,
		"--newline",
		"--no-colors",
		"--no-playlist",
	}
	args = append(args, strategy.ExtraArgs...)
	if strategy.AllowCookies {
		if browser := detectBrowser(); browser != "" {
			args = append(args, "--cookies-from-browser", browser)
		}
	}
	args = append(args, videoURL)
	return args
}

// Download downloads the video at url to outDir using the best available format,
// then calls ExtractMetadata to populate the returned VideoInfo.
// The executor is called at least twice: once for the download, once for metadata.
func (d *YouTubeDownloader) Download(videoURL, outDir string) (VideoInfo, error) {
	videoID := extractVideoID(videoURL)
	outputTemplate := filepath.Join(outDir, videoID+".%(ext)s")
	strategies := downloadStrategies(1080)

	var downloadErr error
	for _, s := range strategies {
		args := buildDownloadArgs(s, outputTemplate, videoURL)
		out, err := d.executor.Run(d.binary, args...)
		if err == nil {
			slog.Debug("download succeeded", "strategy", s.Name)
			break
		}
		slog.Debug("download strategy failed", "strategy", s.Name, "error", err)
		if !isChallenge(out) {
			downloadErr = err
			break
		}
		downloadErr = err
	}
	if downloadErr != nil {
		return VideoInfo{}, downloadErr
	}

	info, err := d.ExtractMetadata(videoURL)
	if err != nil {
		return VideoInfo{}, err
	}
	info.FilePath = filepath.Join(outDir, videoID+".mp4")
	return info, nil
}

// DownloadWithOptions downloads the video at url to outDir with an optional quality override.
// Cache check: if outDir/{videoID}.mp4 already exists, the download step is skipped
// and only ExtractMetadata is called (1 executor call total).
func (d *YouTubeDownloader) DownloadWithOptions(videoURL, outDir, quality string) (VideoInfo, error) {
	info, err := d.ExtractMetadata(videoURL)
	if err != nil {
		return VideoInfo{}, err
	}

	videoID := info.VideoID
	cachedPath := filepath.Join(outDir, videoID+".mp4")

	if fi, statErr := os.Stat(cachedPath); statErr == nil && fi.Size() > 0 {
		info.FilePath = cachedPath
		return info, nil
	}

	qualityInt := 1080
	if quality != "" {
		fmt.Sscanf(quality, "%d", &qualityInt) //nolint:errcheck
	}

	strategies := downloadStrategies(qualityInt)
	outputTemplate := filepath.Join(outDir, videoID+".%(ext)s")

	var downloadErr error
	for _, s := range strategies {
		args := buildDownloadArgs(s, outputTemplate, videoURL)
		out, err := d.executor.Run(d.binary, args...)
		if err == nil {
			slog.Debug("download succeeded", "strategy", s.Name)
			break
		}
		slog.Debug("download strategy failed", "strategy", s.Name, "error", err)
		if !isChallenge(out) {
			downloadErr = err
			break
		}
		downloadErr = err
	}
	if downloadErr != nil {
		return VideoInfo{}, downloadErr
	}

	info.FilePath = cachedPath
	return info, nil
}
