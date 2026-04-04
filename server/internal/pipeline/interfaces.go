package pipeline

import (
	"context"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// Downloader downloads a video from a URL into a local directory.
type Downloader interface {
	DownloadVideo(ctx context.Context, url, outputDir string) (filePath string, err error)
}

// Cutter cuts source video at the given time ranges and returns output clip paths.
type Cutter interface {
	CutSegments(ctx context.Context, sourcePath string, segments []SegmentRange, outputDir string) (clipPaths []string, err error)
}

// SegmentRange is a time range passed to Cutter.CutSegments.
type SegmentRange struct {
	StartSec float64
	EndSec   float64
}

// Transcriber runs speech-to-text on a video file and returns the transcript path.
type Transcriber interface {
	Transcribe(ctx context.Context, sourcePath string) (transcriptPath string, err error)
}

// Analyzer detects highlights in a transcript and returns them.
type Analyzer interface {
	Analyze(ctx context.Context, transcriptPath string) ([]Highlight, error)
}

// ShortsGenerator generates vertical short-form clips and returns their paths.
type ShortsGenerator interface {
	GenerateShorts(ctx context.Context, sourcePath string, segments []SegmentRange, outputDir string) (shortPaths []string, err error)
}

// ThumbnailMaker generates a thumbnail for a given source path and style.
type ThumbnailMaker interface {
	MakeThumbnail(ctx context.Context, sourcePath, style string) (thumbnailPath string, err error)
}

// Uploader uploads a clip to YouTube with the given metadata.
type Uploader interface {
	UploadClip(ctx context.Context, req UploadClipRequest) (youtubeID string, err error)
}

// ChannelConfigReader reads channel configuration for the pipeline.
// Satisfied by *database.ChannelConfigRepo.
type ChannelConfigReader interface {
	GetByChannelIDOrNil(ctx context.Context, channelID int64) (*database.ChannelConfig, error)
}

// UploadClipRequest is passed to Uploader.UploadClip.
type UploadClipRequest struct {
	FilePath      string
	ThumbnailPath string
	Title         string
	Description   string
	Tags          []string
	Privacy       string
}
