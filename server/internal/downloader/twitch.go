package downloader

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// vodIDRe matches the numeric VOD ID embedded in a Twitch URL or a bare ID.
// Kotlin ref: TwitchChatDownloader.VodIdParser + TwitchMetadataExtractor.VodIdParser
var vodIDRe = regexp.MustCompile(`videos/(\d+)`)

// extractVodID returns the numeric VOD ID from a URL like
// https://www.twitch.tv/videos/2642290170 or from a bare "2642290170".
func extractVodID(vodIDOrURL string) string {
	if m := vodIDRe.FindStringSubmatch(vodIDOrURL); m != nil {
		return m[1]
	}
	return vodIDOrURL
}

// TwitchDownloader wraps TwitchDownloaderCLI for VOD and chat downloads.
// Kotlin ref: TwitchDownloader + TwitchVodDownloadStrategy + TwitchChatDownloader.
type TwitchDownloader struct {
	binPath  string
	executor Executor
	retry    RetryConfig
	log      *slog.Logger
}

// NewTwitchDownloader creates a TwitchDownloader.
// Pass an empty string to use the default "TwitchDownloaderCLI" from PATH.
func NewTwitchDownloader(binPath string) *TwitchDownloader {
	if binPath == "" {
		binPath = "TwitchDownloaderCLI"
	}
	return &TwitchDownloader{
		binPath:  binPath,
		executor: &DefaultExecutor{},
		retry:    DefaultRetryConfig,
		log:      slog.With("component", "downloader", "platform", "twitch"),
	}
}

// withExecutor returns a copy with a custom executor (used in tests).
func (d *TwitchDownloader) withExecutor(ex Executor) *TwitchDownloader {
	cp := *d
	cp.executor = ex
	return &cp
}

// DownloadVOD downloads a Twitch VOD to outDir.
// vodID can be a bare numeric ID or a full Twitch URL.
// Kotlin ref: TwitchDownloader.downloadVod() + TwitchDownloaderCliStrategy
func (d *TwitchDownloader) DownloadVOD(vodID, outDir string) (*VideoInfo, error) {
	cleanID := extractVodID(vodID)
	d.log.Info("vod download started", "vodID", cleanID, "outDir", outDir)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create outDir: %w", err)
	}

	outFile := filepath.Join(outDir, cleanID+".mp4")

	_, err := Retry(d.retry, func() (struct{}, error) {
		_, err := d.executor.Run(d.binPath,
			"videodownload",
			"--id", cleanID,
			"-o", outFile,
			"--quality", "1080p60",
			"--threads", "4",
		)
		if err != nil {
			return struct{}{}, fmt.Errorf("TwitchDownloaderCLI videodownload: %w", err)
		}
		return struct{}{}, nil
	})
	if err != nil {
		d.log.Error("vod download failed", "err", err, "vodID", cleanID)
		return nil, fmt.Errorf("download vod %s: %w", cleanID, err)
	}

	info := &VideoInfo{
		VideoID:  cleanID,
		FilePath: outFile,
		Duration: 0 * time.Second, // populated by metadata if needed
	}

	d.log.Info("vod download complete", "vodID", cleanID, "file", outFile)
	return info, nil
}

// DownloadChat downloads only the Twitch chat for a VOD to outDir.
// Output filename: <vodID>_chat.json
// Kotlin ref: TwitchChatDownloader.download()
func (d *TwitchDownloader) DownloadChat(vodID, outDir string) error {
	cleanID := extractVodID(vodID)
	d.log.Info("chat download started", "vodID", cleanID, "outDir", outDir)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create outDir: %w", err)
	}

	outFile := filepath.Join(outDir, cleanID+"_chat.json")

	_, err := Retry(d.retry, func() (struct{}, error) {
		_, err := d.executor.Run(d.binPath,
			"chatdownload",
			"--id", cleanID,
			"-o", outFile,
			"--embed-images",
			"--bttv", "true",
			"--ffz", "true",
			"--stv", "true",
			"--compression", "None",
		)
		if err != nil {
			return struct{}{}, fmt.Errorf("TwitchDownloaderCLI chatdownload: %w", err)
		}
		return struct{}{}, nil
	})
	if err != nil {
		d.log.Error("chat download failed", "err", err, "vodID", cleanID)
		return fmt.Errorf("download chat %s: %w", cleanID, err)
	}

	d.log.Info("chat download complete", "vodID", cleanID, "file", outFile)
	return nil
}
