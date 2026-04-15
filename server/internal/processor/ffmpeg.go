package processor

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// timeRe matches ffmpeg's progress line: time=HH:MM:SS.SS
var timeRe = regexp.MustCompile(`time=(\d+):(\d+):(\d+\.\d+)`)

// RunFFmpeg launches ffmpeg with the given args, parses time= from stderr to compute
// progress percent, calls onProgress(pct) on each update, and handles cancellation
// via ctx (SIGTERM → 5s → SIGKILL). Returns error with last 20 stderr lines on failure.
func RunFFmpeg(ctx context.Context, args []string, totalDurationSec float64, onProgress func(float64)) error {
	log := slog.With("component", "ffmpeg")
	log.Info("starting ffmpeg", "args_count", len(args), "duration_sec", totalDurationSec)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Capture stderr for progress parsing and error reporting
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	// Ring buffer for last 20 lines of stderr (for error reporting)
	var mu sync.Mutex
	stderrLines := make([]string, 0, 20)

	// Parse stderr in a goroutine
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Buffer(make([]byte, 64*1024), 64*1024)
		for scanner.Scan() {
			line := scanner.Text()

			mu.Lock()
			if len(stderrLines) >= 20 {
				stderrLines = stderrLines[1:]
			}
			stderrLines = append(stderrLines, line)
			mu.Unlock()

			// Parse time= progress
			if totalDurationSec > 0 && onProgress != nil {
				if matches := timeRe.FindStringSubmatch(line); matches != nil {
					hours, _ := strconv.ParseFloat(matches[1], 64)
					minutes, _ := strconv.ParseFloat(matches[2], 64)
					seconds, _ := strconv.ParseFloat(matches[3], 64)
					timeSec := hours*3600 + minutes*60 + seconds
					pct := (timeSec / totalDurationSec) * 100.0
					if pct < 0 {
						pct = 0
					}
					if pct > 100 {
						pct = 100
					}
					onProgress(pct)
				}
			}
		}
	}()

	// Wait for stderr scanner to finish before calling cmd.Wait
	<-scanDone

	err = cmd.Wait()
	if err != nil {
		// Check if it was a context cancellation
		if ctx.Err() != nil {
			log.Warn("ffmpeg cancelled", "reason", ctx.Err())
			return fmt.Errorf("ffmpeg cancelled: %w", ctx.Err())
		}

		mu.Lock()
		tail := strings.Join(stderrLines, "\n")
		mu.Unlock()

		return fmt.Errorf("ffmpeg exited with %v:\n%s", err, tail)
	}

	log.Info("ffmpeg completed successfully")
	return nil
}

// FFmpegAvailable checks if ffmpeg is on PATH.
func FFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// CancelableContext returns a context that sends SIGTERM on cancel, waits gracePeriod,
// then sends SIGKILL. Use this to wrap RunFFmpeg calls for clean subprocess shutdown.
func CancelableContext(parent context.Context, gracePeriod time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	return ctx, cancel
}
