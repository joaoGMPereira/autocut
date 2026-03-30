package downloader

import (
	"errors"
	"testing"
	"time"
)

// zeroDelayConfig disables sleep so tests run instantly.
var zeroDelayConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 0,
}

func TestRetry_ImmediateSuccess(t *testing.T) {
	calls := 0
	result, err := Retry(zeroDelayConfig, func() (int, error) {
		calls++
		return 42, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetry_TwoFailuresThenSuccess(t *testing.T) {
	calls := 0
	sentinel := errors.New("transient error")

	result, err := Retry(zeroDelayConfig, func() (string, error) {
		calls++
		if calls < 3 {
			return "", sentinel
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected \"ok\", got %q", result)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_AllFailures_ReturnsLastError(t *testing.T) {
	sentinel := errors.New("permanent failure")
	calls := 0

	_, err := Retry(zeroDelayConfig, func() (bool, error) {
		calls++
		return false, sentinel
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected error chain to contain sentinel, got: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}
	calls := 0
	start := time.Now()

	_, _ = Retry(cfg, func() (struct{}, error) {
		calls++
		return struct{}{}, errors.New("always fails")
	})

	elapsed := time.Since(start)
	// Expected sleeps: 10ms + 20ms = 30ms minimum.
	minExpected := 25 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("expected at least %v elapsed (backoff), got %v", minExpected, elapsed)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}
