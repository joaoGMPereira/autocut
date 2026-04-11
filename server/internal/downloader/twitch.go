package downloader

import (
	"encoding/json"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/progress"
)

// TwitchDownloader downloads and inspects Twitch VODs/clips using yt-dlp.
type TwitchDownloader struct {
	binary   string
	executor Executor
	retry    RetryConfig
	reporter progress.Reporter
	jobID    string
}

// NewTwitchDownloader creates a downloader backed by the given binary name.
// If binary is empty, it defaults to "yt-dlp".
func NewTwitchDownloader(binary string) *TwitchDownloader {
	if binary == "" {
		binary = "yt-dlp"
	}
	return &TwitchDownloader{
		binary:   binary,
		executor: OSExecutor{},
		retry:    RetryConfig{MaxAttempts: 3, InitialDelay: time.Second},
		reporter: progress.NoopReporter{},
	}
}

// WithReporter injects a progress reporter. Returns receiver for chaining.
func (d *TwitchDownloader) WithReporter(r progress.Reporter) *TwitchDownloader {
	d.reporter = r
	return d
}

// WithJobID sets the job ID used when emitting progress events. Returns receiver for chaining.
func (d *TwitchDownloader) WithJobID(id string) *TwitchDownloader {
	d.jobID = id
	return d
}

// ExtractMetadata calls yt-dlp --dump-json --no-download on a Twitch URL.
func (d *TwitchDownloader) ExtractMetadata(rawURL string) (VideoInfo, error) {
	d.reporter.Report(d.jobID, progress.Event{Stage: "metadata", Percent: -1})
	return Retry(d.retry, func() (VideoInfo, error) {
		out, err := d.executor.Run(d.binary, "--dump-json", "--no-download", rawURL)
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

// DownloadWithOptions downloads a Twitch VOD/clip to outDir with optional quality override.
func (d *TwitchDownloader) DownloadWithOptions(videoURL, outDir, quality string) (VideoInfo, error) {
	info, err := d.ExtractMetadata(videoURL)
	if err != nil {
		return VideoInfo{}, err
	}

	_ = quality

	args := []string{
		"--format", "best",
		"--output", outDir + "/%(id)s.%(ext)s",
		"--no-playlist",
		videoURL,
	}
	if _, err := d.executor.Run(d.binary, args...); err != nil {
		d.reporter.Report(d.jobID, progress.Event{Stage: "error", Message: err.Error(), Percent: -1})
		return VideoInfo{}, err
	}

	d.reporter.Report(d.jobID, progress.Event{Stage: "done", Percent: 100})
	return info, nil
}
