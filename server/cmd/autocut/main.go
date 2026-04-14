package main

import (
	"fmt"
	"os"

	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/progress"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "autocut",
	Short: "AutoCut CLI",
}

var downloadCmd = &cobra.Command{
	Use:   "download <url> <output-dir>",
	Short: "Download a YouTube video",
	Args:  cobra.ExactArgs(2),
	RunE:  runDownload,
}

var quality string

func init() {
	downloadCmd.Flags().StringVar(&quality, "quality", "", "video quality (e.g. 1080, 720); defaults to 1080")
	rootCmd.AddCommand(downloadCmd)
}

func runDownload(cmd *cobra.Command, args []string) error {
	videoURL := args[0]
	outDir := args[1]

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	d := downloader.NewYouTubeDownloader("yt-dlp").
		WithReporter(progress.NewWriterReporter(os.Stderr))

	fmt.Printf("Downloading %s → %s\n", videoURL, outDir)
	info, err := d.DownloadWithOptions(videoURL, outDir, quality)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("Done.\n")
	fmt.Printf("  Title:    %s\n", info.Title)
	fmt.Printf("  VideoID:  %s\n", info.VideoID)
	fmt.Printf("  Duration: %s\n", info.Duration)
	fmt.Printf("  File:     %s\n", info.FilePath)
	return nil
}
