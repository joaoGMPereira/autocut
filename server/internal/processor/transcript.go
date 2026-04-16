package processor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TranscribeFullVideo extracts a full-video WAV and runs whisper-cli on it.
// Returns the path to the whisper-cli -oj output JSON file (persisted, not a temp file).
// Also returns a cleanup func that removes the temporary WAV file — caller must invoke it when done.
// onProgress receives fake-linear progress values in [0, 100] during transcription.
// Estimated processing duration = max(30, durationSec * 1.5) seconds.
func TranscribeFullVideo(
	videoPath string,
	modelPath string,
	dataDir string,
	durationSec int64,
	onProgress func(pct float64),
	ctx context.Context,
) (transcriptJSONPath string, cleanup func(), err error) {
	pid := os.Getpid()
	audioPath := filepath.Join(os.TempDir(), fmt.Sprintf("exec_audio_%d.wav", pid))
	// Whisper JSON output lives in dataDir/transcripts so it persists as transcript_path.
	whisperOutDir := filepath.Join(dataDir, "transcripts")
	if mkErr := os.MkdirAll(whisperOutDir, 0o755); mkErr != nil {
		return "", func() {}, fmt.Errorf("create transcript dir: %w", mkErr)
	}
	whisperOutBase := filepath.Join(whisperOutDir, fmt.Sprintf("transcript_%d", pid))
	transcriptOut := whisperOutBase + ".json"

	cleanupFn := func() {
		_ = os.Remove(audioPath)
	}

	slog.Info("TranscribeFullVideo: extracting WAV", "video_path", videoPath, "audio_path", audioPath)

	// Extract 16 kHz mono WAV — whisper.cpp requires WAV input.
	// Full-video variant: no -ss or -t args (no time window).
	audioArgs := []string{
		"-y",
		"-i", videoPath,
		"-vn", "-ar", "16000", "-ac", "1", "-f", "wav",
		audioPath,
	}
	if out, cmdErr := exec.Command("ffmpeg", audioArgs...).CombinedOutput(); cmdErr != nil {
		return "", cleanupFn, fmt.Errorf("extract audio for whisper: %w: %s", cmdErr, out)
	}

	slog.Info("TranscribeFullVideo: WAV extracted, starting whisper-cli", "model", modelPath)

	// Estimate processing time for fake-linear progress ticker.
	estimatedSec := float64(durationSec) * 1.5
	if estimatedSec < 30 {
		estimatedSec = 30
	}

	// Run whisper-cli; -l auto lets it detect the language.
	whisperArgs := []string{
		"-m", modelPath,
		"-oj", "-of", whisperOutBase,
		"-l", "auto",
		audioPath,
	}
	cmd := exec.CommandContext(ctx, "whisper-cli", whisperArgs...)

	startTime := time.Now()

	// stopTicker signals the progress goroutine to exit when whisper-cli finishes.
	stopTicker := make(chan struct{})

	// Ticker goroutine: emits fake-linear progress every 2s while whisper runs.
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds()
				pct := elapsed / estimatedSec * 100
				if pct > 95 {
					pct = 95
				}
				if onProgress != nil {
					onProgress(pct)
				}
			case <-stopTicker:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Run whisper-cli synchronously.
	out, cmdErr := cmd.CombinedOutput()

	// Signal ticker to stop regardless of outcome.
	close(stopTicker)

	if cmdErr != nil {
		return "", cleanupFn, fmt.Errorf("whisper-cli: %w: %s", cmdErr, out)
	}

	slog.Info("TranscribeFullVideo: whisper-cli complete", "output_json", transcriptOut)

	// Verify the output JSON was created.
	if _, statErr := os.Stat(transcriptOut); statErr != nil {
		return "", cleanupFn, fmt.Errorf("whisper output JSON not found at %s: %w", transcriptOut, statErr)
	}

	// Emit 100% on success.
	if onProgress != nil {
		onProgress(100)
	}

	return transcriptOut, cleanupFn, nil
}
