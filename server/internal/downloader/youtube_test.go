package downloader

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// MockExecutor records all calls and returns configured output/error.
type MockExecutor struct {
	// calls stores every (name, args) invocation.
	calls []mockCall
	// response is returned by Run.
	response []byte
	// err is returned by Run when non-nil.
	err error
}

type mockCall struct {
	name string
	args []string
}

func (m *MockExecutor) Run(name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, mockCall{name: name, args: args})
	return m.response, m.err
}

// ytDlpFixture is a minimal yt-dlp --dump-json payload used in tests.
const ytDlpFixture = `{
  "id": "dQw4w9WgXcQ",
  "title": "Rick Astley - Never Gonna Give You Up",
  "description": "The official video.",
  "duration": 212.5,
  "thumbnail": "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
  "_filename": ""
}`

// TestExtractMetadata_ParsesJSON verifies that ExtractMetadata correctly
// deserialises the yt-dlp JSON fixture into a VideoInfo.
func TestExtractMetadata_ParsesJSON(t *testing.T) {
	mock := &MockExecutor{response: []byte(ytDlpFixture)}
	d := NewYouTubeDownloader("").withExecutor(mock)
	d.retry = RetryConfig{MaxAttempts: 1, InitialDelay: 0} // single attempt

	info, err := d.ExtractMetadata("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.VideoID != "dQw4w9WgXcQ" {
		t.Errorf("VideoID: want dQw4w9WgXcQ, got %q", info.VideoID)
	}
	if info.Title != "Rick Astley - Never Gonna Give You Up" {
		t.Errorf("Title mismatch: got %q", info.Title)
	}
	if info.Description != "The official video." {
		t.Errorf("Description mismatch: got %q", info.Description)
	}
	if info.ThumbnailURL != "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg" {
		t.Errorf("ThumbnailURL mismatch: got %q", info.ThumbnailURL)
	}

	wantDuration := time.Duration(212.5 * float64(time.Second))
	if info.Duration != wantDuration {
		t.Errorf("Duration: want %v, got %v", wantDuration, info.Duration)
	}
}

// TestExtractMetadata_CorrectArgs verifies the exact yt-dlp flags forwarded
// to the executor.
func TestExtractMetadata_CorrectArgs(t *testing.T) {
	mock := &MockExecutor{response: []byte(ytDlpFixture)}
	d := NewYouTubeDownloader("yt-dlp").withExecutor(mock)
	d.retry = RetryConfig{MaxAttempts: 1, InitialDelay: 0}

	const testURL = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	_, err := d.ExtractMetadata(testURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) == 0 {
		t.Fatal("executor was never called")
	}

	call := mock.calls[0]
	if call.name != "yt-dlp" {
		t.Errorf("binary name: want yt-dlp, got %q", call.name)
	}

	wantArgs := []string{"--dump-json", "--no-download", "--no-playlist", testURL}
	if len(call.args) != len(wantArgs) {
		t.Fatalf("args length: want %d, got %d: %v", len(wantArgs), len(call.args), call.args)
	}
	for i, want := range wantArgs {
		if call.args[i] != want {
			t.Errorf("arg[%d]: want %q, got %q", i, want, call.args[i])
		}
	}
}

// TestDownload_CallsExecutorWithCorrectFlags ensures Download passes the right
// flags when the executor succeeds, and that it calls ExtractMetadata afterwards.
func TestDownload_CallsExecutorWithCorrectFlags(t *testing.T) {
	mock := &MockExecutor{response: []byte(ytDlpFixture)}
	d := NewYouTubeDownloader("yt-dlp").withExecutor(mock)
	d.retry = RetryConfig{MaxAttempts: 1, InitialDelay: 0}

	outDir := t.TempDir()
	const testURL = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	_, err := d.Download(testURL, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Download should call executor at least twice: once for the video download
	// and once for ExtractMetadata.
	if len(mock.calls) < 2 {
		t.Fatalf("expected at least 2 executor calls, got %d", len(mock.calls))
	}

	// First call must be the download invocation.
	first := mock.calls[0]
	if first.name != "yt-dlp" {
		t.Errorf("first call binary: want yt-dlp, got %q", first.name)
	}
	// Must contain --format flag.
	hasFormat := false
	for _, a := range first.args {
		if strings.Contains(a, "bestvideo") {
			hasFormat = true
		}
	}
	if !hasFormat {
		t.Errorf("expected --format with bestvideo in first call args: %v", first.args)
	}
}

// TestExtractMetadata_ReturnsErrorOnExecutorFailure verifies retry exhaustion
// is surfaced to the caller.
func TestExtractMetadata_ReturnsErrorOnExecutorFailure(t *testing.T) {
	mock := &MockExecutor{err: errors.New("yt-dlp not found")}
	d := NewYouTubeDownloader("").withExecutor(mock)
	d.retry = RetryConfig{MaxAttempts: 2, InitialDelay: 0}

	_, err := d.ExtractMetadata("https://www.youtube.com/watch?v=xyz")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
