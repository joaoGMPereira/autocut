package progress_test

import (
	"testing"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/progress"
)

func TestReport_SSEReporter(t *testing.T) {
	h := hub.New()
	ch, cancel := h.Register("j1")
	defer cancel()

	r := progress.NewSSEReporter(h)
	r.Report("j1", progress.Event{Stage: "download", Message: "android_client", Percent: -1})

	select {
	case received := <-ch:
		if received.Type != "download" {
			t.Errorf("Type: want %q, got %q", "download", received.Type)
		}
		data, ok := received.Data.(progress.Event)
		if !ok {
			t.Fatalf("Data: expected progress.Event, got %T", received.Data)
		}
		if data.Message != "android_client" {
			t.Errorf("Message: want %q, got %q", "android_client", data.Message)
		}
		if data.Percent != -1 {
			t.Errorf("Percent: want -1, got %v", data.Percent)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout: no SSE event received")
	}
}

func TestNoopReporter_DoesNotBlock(t *testing.T) {
	r := progress.NoopReporter{}
	// Must return immediately — no panic, no block
	done := make(chan struct{})
	go func() {
		r.Report("any-job", progress.Event{Stage: "done", Percent: 100})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("NoopReporter.Report blocked")
	}
}
