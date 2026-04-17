package thumbnail

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/joaoGMPereira/autocut/server/internal/processor"
)

// RunFFmpeg delegates to processor.RunFFmpeg with thumbnail-specific logging.
// totalDurationSec can be 0 for single-frame extraction (no progress reporting).
func RunFFmpeg(ctx context.Context, args []string, totalDurationSec float64, onProgress func(float64)) error {
	slog.With("component", "thumbnail.executor").
		Info("running ffmpeg for thumbnail", "args_count", len(args))
	return processor.RunFFmpeg(ctx, args, totalDurationSec, onProgress)
}

// CheckFFmpegAvailable returns an error if ffmpeg is not on PATH.
func CheckFFmpegAvailable() error {
	if !processor.FFmpegAvailable() {
		return fmt.Errorf("ffmpeg not found on PATH — install ffmpeg to generate thumbnails")
	}
	return nil
}

// ImageMagickBinary returns the ImageMagick binary name.
// Checks for "magick" (v7) first, falls back to "convert" (v6).
func ImageMagickBinary() string {
	if _, err := exec.LookPath("magick"); err == nil {
		return "magick"
	}
	if _, err := exec.LookPath("convert"); err == nil {
		return "convert"
	}
	return "magick" // default, will fail at execution
}

// RunImageMagick executes an ImageMagick command with the given args.
func RunImageMagick(ctx context.Context, args []string) error {
	slog.With("component", "thumbnail.executor").
		Info("running imagemagick for thumbnail", "args_count", len(args))

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("imagemagick failed: %w\noutput: %s", err, string(output))
	}
	return nil
}

// ExtractFrame extracts a single frame from a video at seekSec and saves it as a JPEG.
// Used as a fallback when no base image is available for landscape thumbnails.
func ExtractFrame(ctx context.Context, videoPath string, seekSec float64, outputPath string) error {
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.2f", seekSec),
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "2",
		outputPath,
	}
	return RunFFmpeg(ctx, args, 0, nil)
}

// CheckImageMagickAvailable returns an error if neither magick nor convert is on PATH.
func CheckImageMagickAvailable() error {
	if _, err := exec.LookPath("magick"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("convert"); err == nil {
		return nil
	}
	return fmt.Errorf("ImageMagick not found on PATH — install ImageMagick to generate landscape thumbnails")
}
