package api_test

import (
	"testing"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

func TestSSERegisterPublish(t *testing.T) {
	h := hub.New()

	ch, cancel := h.Register("job-1")
	defer cancel()

	event := hub.SSEEvent{Type: "progress", Data: map[string]string{"status": "ok"}}
	h.Publish("job-1", event)

	select {
	case got := <-ch:
		if got.Type != event.Type {
			t.Errorf("expected type %q, got %q", event.Type, got.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout: no event received")
	}
}

func TestSSEMultipleListeners(t *testing.T) {
	h := hub.New()

	ch1, cancel1 := h.Register("job-multi")
	defer cancel1()

	ch2, cancel2 := h.Register("job-multi")
	defer cancel2()

	event := hub.SSEEvent{Type: "done", Data: nil}
	h.Publish("job-multi", event)

	for i, ch := range []<-chan hub.SSEEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Type != "done" {
				t.Errorf("listener %d: expected type %q, got %q", i+1, "done", got.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("listener %d: timeout — event not received", i+1)
		}
	}
}

func TestSSECancel(t *testing.T) {
	h := hub.New()

	_, cancel := h.Register("job-cancel")
	cancel() // cancel immediately before publishing

	// Publish after cancel should not block and should not panic
	done := make(chan struct{})
	go func() {
		h.Publish("job-cancel", hub.SSEEvent{Type: "ping"})
		close(done)
	}()

	select {
	case <-done:
		// success: publish returned without blocking
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout: Publish blocked after cancel")
	}
}
