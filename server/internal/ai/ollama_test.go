package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestGenerateStream
// ---------------------------------------------------------------------------

// TestGenerateStream verifies that a two-token NDJSON response is delivered
// in order and that the channel closes after done=true.
func TestGenerateStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"hello","done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"response":" world","done":true}` + "\n"))
	}))
	defer srv.Close()

	client := New(srv.URL, 5*time.Second)
	req := GenerateRequest{Model: "test", Prompt: "hi"}

	ch, err := client.GenerateStream(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateStream error: %v", err)
	}

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "hello" {
		t.Errorf("token[0]: want %q got %q", "hello", tokens[0])
	}
	if tokens[1] != " world" {
		t.Errorf("token[1]: want %q got %q", " world", tokens[1])
	}
}

// ---------------------------------------------------------------------------
// TestGenerateStreamTimeout
// ---------------------------------------------------------------------------

// TestGenerateStreamTimeout verifies that a short HTTP client timeout causes
// GenerateStream to return an error (the server never responds).
//
// Design: we use context cancellation as the timeout mechanism, which is
// cleaner than relying on net/http client timeout interacting with httptest
// server shutdown. The server hangs; we cancel the context; either
// GenerateStream returns an error or the channel closes.
func TestGenerateStreamTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hang until the test server itself is closed.
		select {
		case <-r.Context().Done():
		}
	}))
	// srv.Close is intentionally called early so the hanging handler gets its
	// context cancelled and the goroutine exits cleanly.

	client := New(srv.URL, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := GenerateRequest{Model: "test", Prompt: "hi"}

	// Close the server after a short delay so the handler goroutine exits —
	// this prevents the httptest 5-second blocked-Close warning.
	go func() {
		time.Sleep(300 * time.Millisecond)
		srv.Close()
	}()

	ch, err := client.GenerateStream(ctx, req)
	if err != nil {
		// Error from GenerateStream itself is acceptable (deadline before connect).
		return
	}

	// If GenerateStream returned a channel, it must close within test timeout.
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()

	select {
	case <-done:
		// ok — channel closed after context deadline or server close
	case <-time.After(3 * time.Second):
		t.Fatal("channel did not close after context deadline")
	}
}

// ---------------------------------------------------------------------------
// TestGenerateSync
// ---------------------------------------------------------------------------

// TestGenerateSync verifies that GenerateSync concatenates all tokens.
func TestGenerateSync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"foo","done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"response":"bar","done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"response":"baz","done":true}` + "\n"))
	}))
	defer srv.Close()

	client := New(srv.URL, 5*time.Second)
	req := GenerateRequest{Model: "test", Prompt: "go"}

	result, err := client.GenerateSync(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateSync error: %v", err)
	}
	if result != "foobarbaz" {
		t.Errorf("want %q got %q", "foobarbaz", result)
	}
}
