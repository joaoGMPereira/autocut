//go:build integration

package integration_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/ai"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/transcript"
)

// ---------------------------------------------------------------------------
// TestRetryLogic — exponential backoff 3x
// ---------------------------------------------------------------------------

// TestRetryLogic verifies that Retry calls fn exactly 3 times when it fails
// twice and succeeds on the third attempt, proving the exponential backoff path.
func TestRetryLogic(t *testing.T) {
	cfg := downloader.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond, // fast for CI
	}

	calls := 0
	result, err := downloader.Retry(cfg, func() (string, error) {
		calls++
		if calls < 3 {
			return "", &testError{"transient failure attempt " + itoa(calls)}
		}
		return "success", nil
	})

	if err != nil {
		t.Fatalf("expected nil error after 3 attempts, got: %v", err)
	}
	if result != "success" {
		t.Errorf("want result %q, got %q", "success", result)
	}
	if calls != 3 {
		t.Errorf("want 3 calls (2 failures + 1 success), got %d", calls)
	}
}

// ---------------------------------------------------------------------------
// TestSSEKeepalive — ping arrives within 35s (uses 30s ticker)
// ---------------------------------------------------------------------------

// TestSSEKeepalive registers a listener via ServeSSE and verifies that a
// keepalive "ping" event arrives within 35 seconds (hub sends every 30s).
func TestSSEKeepalive(t *testing.T) {
	h := hub.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeSSE(w, r, "keepalive-job")
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("want Content-Type text/event-stream, got %q", resp.Header.Get("Content-Type"))
	}

	// Read body lines until we see a ping or the context expires.
	pingReceived := make(chan struct{}, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, `"ping"`) {
				pingReceived <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-pingReceived:
		// pass
	case <-ctx.Done():
		t.Fatal("timeout: no ping received within 35s")
	}
}

// ---------------------------------------------------------------------------
// TestWhisperJSONParsing — parse inline fixture + testdata file
// ---------------------------------------------------------------------------

// TestWhisperJSONParsing verifies the Whisper JSON parser with an inline
// fixture and with the real testdata/whisper_sample.json file.
func TestWhisperJSONParsing(t *testing.T) {
	t.Run("inline_two_segments", func(t *testing.T) {
		fixture := []byte(`{
			"segments": [
				{"start": 0.0,  "end": 2.5,  "text": "Hello world"},
				{"start": 2.5,  "end": 5.0,  "text": "This is a test"}
			]
		}`)

		tr, err := transcript.ParseWhisperJSON(fixture)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if len(tr.Segments) != 2 {
			t.Fatalf("want 2 segments, got %d", len(tr.Segments))
		}

		// Segment 0 timestamps
		wantStart0 := time.Duration(0)
		wantEnd0 := time.Duration(2.5 * float64(time.Second))
		if tr.Segments[0].Start != wantStart0 {
			t.Errorf("seg[0].Start: want %v, got %v", wantStart0, tr.Segments[0].Start)
		}
		if tr.Segments[0].End != wantEnd0 {
			t.Errorf("seg[0].End: want %v, got %v", wantEnd0, tr.Segments[0].End)
		}
		if tr.Segments[0].Text != "Hello world" {
			t.Errorf("seg[0].Text: want %q, got %q", "Hello world", tr.Segments[0].Text)
		}

		// Segment 1 timestamps
		wantStart1 := time.Duration(2.5 * float64(time.Second))
		wantEnd1 := time.Duration(5.0 * float64(time.Second))
		if tr.Segments[1].Start != wantStart1 {
			t.Errorf("seg[1].Start: want %v, got %v", wantStart1, tr.Segments[1].Start)
		}
		if tr.Segments[1].End != wantEnd1 {
			t.Errorf("seg[1].End: want %v, got %v", wantEnd1, tr.Segments[1].End)
		}
		if tr.Segments[1].Text != "This is a test" {
			t.Errorf("seg[1].Text: want %q, got %q", "This is a test", tr.Segments[1].Text)
		}
	})

	t.Run("testdata_three_segments", func(t *testing.T) {
		data, err := os.ReadFile("testdata/whisper_sample.json")
		if err != nil {
			t.Fatalf("read testdata: %v", err)
		}

		tr, err := transcript.ParseWhisperJSON(data)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if len(tr.Segments) != 3 {
			t.Fatalf("want 3 segments, got %d", len(tr.Segments))
		}
		if tr.Language != "en" {
			t.Errorf("want language %q, got %q", "en", tr.Language)
		}
		// Last segment ends at 12.4s
		wantDuration := time.Duration(12.4 * float64(time.Second))
		if tr.Duration != wantDuration {
			t.Errorf("Duration: want %v, got %v", wantDuration, tr.Duration)
		}
	})
}

// ---------------------------------------------------------------------------
// TestOllamaStreaming — NDJSON mock server, 5 tokens
// ---------------------------------------------------------------------------

// TestOllamaStreaming verifies that OllamaClient.GenerateSync correctly
// reads a 5-token NDJSON stream from a mock server and returns the
// concatenated result.
func TestOllamaStreaming(t *testing.T) {
	tokens := []string{"The ", "quick ", "brown ", "fox ", "jumps"}
	ndjson := ""
	for i, tok := range tokens {
		done := i == len(tokens)-1
		if done {
			ndjson += `{"response":"` + tok + `","done":true}` + "\n"
		} else {
			ndjson += `{"response":"` + tok + `","done":false}` + "\n"
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %q", r.Method)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ndjson))
	}))
	defer srv.Close()

	client := ai.New(srv.URL, 5*time.Second)
	req := ai.GenerateRequest{
		Model:  "qwen2.5:7b",
		Prompt: "tell me something",
	}

	ctx := context.Background()
	result, err := client.GenerateSync(ctx, req)
	if err != nil {
		t.Fatalf("GenerateSync error: %v", err)
	}

	want := "The quick brown fox jumps"
	if result != want {
		t.Errorf("want %q, got %q", want, result)
	}
}

// ---------------------------------------------------------------------------
// TestThumbnailCacheRoundTrip — end-to-end cache with httptest PNG server
// ---------------------------------------------------------------------------

// TestThumbnailCacheRoundTrip serves a fake PNG image over httptest, calls
// Download, verifies the file is written to t.TempDir(), then calls Get to
// confirm the cache entry was persisted correctly.
func TestThumbnailCacheRoundTrip(t *testing.T) {
	// Minimal 1x1 PNG (67 bytes — valid PNG header + IDAT + IEND)
	fakePNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakePNG)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	cache := downloader.NewThumbnailCache(tmpDir)

	imageURL := srv.URL + "/thumbnail.png"

	// First call: should download and cache.
	path, err := cache.Download(imageURL, tmpDir)
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	if path == "" {
		t.Fatal("Download returned empty path")
	}

	// File must exist on disk.
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("downloaded file does not exist at %q: %v", path, statErr)
	}

	// Verify file content matches what the server sent.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if len(data) != len(fakePNG) {
		t.Errorf("file size mismatch: want %d bytes, got %d bytes", len(fakePNG), len(data))
	}

	// Get must now return a cache hit.
	cachedPath, ok := cache.Get(imageURL)
	if !ok {
		t.Fatal("Get returned cache miss after Download")
	}
	if cachedPath != path {
		t.Errorf("cached path mismatch: want %q, got %q", path, cachedPath)
	}

	// Second call: should return cached path without making HTTP request.
	// We close the server to confirm no HTTP is attempted.
	srv.Close()
	path2, err := cache.Download(imageURL, tmpDir)
	if err != nil {
		t.Fatalf("second Download (cached) error: %v", err)
	}
	if path2 != path {
		t.Errorf("second Download returned different path: want %q, got %q", path, path2)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// itoa is a minimal int→string for test helpers (avoids importing strconv/fmt).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 4)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
