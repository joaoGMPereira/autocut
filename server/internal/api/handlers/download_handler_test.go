package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/downloader"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

func TestPostDownloadReturnsJobID(t *testing.T) {
	h := hub.New()
	ytDl := downloader.NewYouTubeDownloader("")
	twDl := downloader.NewTwitchDownloader("")
	handler := handlers.NewDownloadHandler(h, ytDl, twDl)

	body := `{"url":"https://youtu.be/test","type":"youtube","output_dir":"/tmp/test-dl"}`
	req := httptest.NewRequest(http.MethodPost, "/api/download", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.PostDownload(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	jobID, ok := resp["job_id"]
	if !ok || jobID == "" {
		t.Errorf("expected non-empty job_id in response, got: %v", resp)
	}
}

func TestGetDownloadStreamHeaders(t *testing.T) {
	h := hub.New()
	ytDl := downloader.NewYouTubeDownloader("")
	twDl := downloader.NewTwitchDownloader("")
	handler := handlers.NewDownloadHandler(h, ytDl, twDl)

	jobID := "test-stream-id"

	// Use a cancelled context so ServeSSE returns immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling — ServeSSE exits on r.Context().Done()

	req := httptest.NewRequest(http.MethodGet, "/api/download/"+jobID+"/stream", nil)
	req = req.WithContext(ctx)
	req.SetPathValue("id", jobID)

	rec := httptest.NewRecorder()
	handler.GetDownloadStream(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
}
