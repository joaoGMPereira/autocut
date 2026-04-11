package downloader

import "time"

// RetryConfig controls how many times an operation is attempted and how long
// to wait between failures.
type RetryConfig struct {
	// MaxAttempts is the total number of tries (including the first).
	MaxAttempts int

	// InitialDelay is the wait before the second attempt.
	// Each subsequent sleep doubles: InitialDelay * 2^(attempt-1).
	// Zero disables sleeping entirely (useful in tests).
	InitialDelay time.Duration
}

// Retry calls fn up to cfg.MaxAttempts times with exponential back-off.
// It returns the first successful result or the last error if all attempts fail.
//
// Backoff schedule (i is the zero-based iteration index):
//
//	i=0: no sleep
//	i=1: sleep InitialDelay * 2^0 = InitialDelay
//	i=2: sleep InitialDelay * 2^1 = 2 * InitialDelay
func Retry[T any](cfg RetryConfig, fn func() (T, error)) (T, error) {
	var lastErr error
	var zero T
	for i := 0; i < cfg.MaxAttempts; i++ {
		if i > 0 && cfg.InitialDelay > 0 {
			delay := cfg.InitialDelay * (1 << uint(i-1))
			time.Sleep(delay)
		}
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return zero, lastErr
}
