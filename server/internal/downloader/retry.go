package downloader

import (
	"fmt"
	"log/slog"
	"time"
)

// RetryConfig controls exponential-backoff retry behaviour.
type RetryConfig struct {
	// MaxAttempts is the total number of tries (including the first attempt).
	MaxAttempts int

	// InitialDelay is the wait duration before the second attempt.
	// Each subsequent delay doubles (100ms → 200ms → 400ms).
	InitialDelay time.Duration
}

// DefaultRetryConfig is the standard retry policy used by all downloaders:
// 3 attempts with delays of 100 ms and 200 ms between them.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 100 * time.Millisecond,
}

// Retry executes fn up to cfg.MaxAttempts times, applying exponential backoff
// between failures. Returns the first successful result or the last error.
//
// Backoff schedule (default config):
//
//	attempt 1 → run immediately
//	attempt 2 → wait 100 ms
//	attempt 3 → wait 200 ms
func Retry[T any](cfg RetryConfig, fn func() (T, error)) (T, error) {
	logger := slog.With("component", "downloader", "op", "retry")

	var zero T
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		if attempt == cfg.MaxAttempts {
			return zero, fmt.Errorf("failed after %d attempts: %w", cfg.MaxAttempts, err)
		}

		logger.Warn("retry attempt",
			"attempt", attempt,
			"maxAttempts", cfg.MaxAttempts,
			"delay", delay,
			"err", err,
		)

		time.Sleep(delay)
		delay *= 2
	}

	// Unreachable — loop always returns via the maxAttempts branch above.
	return zero, fmt.Errorf("retry: unexpected exit")
}
