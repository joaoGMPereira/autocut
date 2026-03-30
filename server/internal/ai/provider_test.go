package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// mockProvider — implements AIProvider for tests
// ---------------------------------------------------------------------------

type mockProvider struct {
	name      string
	available bool
}

func (m *mockProvider) Name() string        { return m.name }
func (m *mockProvider) IsAvailable() bool   { return m.available }
func (m *mockProvider) GenerateSync(_ context.Context, _ GenerateRequest) (string, error) {
	return "mock", nil
}
func (m *mockProvider) GenerateStream(_ context.Context, _ GenerateRequest) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "mock"
	close(ch)
	return ch, nil
}

// ---------------------------------------------------------------------------
// OllamaProvider availability tests
// ---------------------------------------------------------------------------

// TestOllamaProviderAvailable verifies that IsAvailable() returns true when
// the server responds with HTTP 200.
func TestOllamaProviderAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, 2*time.Second)
	if !p.IsAvailable() {
		t.Error("expected IsAvailable() == true for a live server")
	}
}

// TestOllamaProviderUnavailable verifies that IsAvailable() returns false when
// the connection is refused.
func TestOllamaProviderUnavailable(t *testing.T) {
	// Port 19999 is almost certainly closed; connection will be refused.
	p := NewOllamaProvider("http://localhost:19999", 2*time.Second)
	if p.IsAvailable() {
		t.Error("expected IsAvailable() == false for a closed port")
	}
}

// ---------------------------------------------------------------------------
// ProviderFactory tests
// ---------------------------------------------------------------------------

// TestProviderFactoryDefault verifies that Default() returns the preferred
// provider when it is available.
func TestProviderFactoryDefault(t *testing.T) {
	f := &ProviderFactory{
		providers: make(map[string]AIProvider),
	}
	mock := &mockProvider{name: "test", available: true}
	f.Register(mock)
	f.SetPreferred("test")

	got := f.Default()
	if got == nil {
		t.Fatal("expected non-nil default provider")
	}
	if got.Name() != "test" {
		t.Errorf("expected provider name %q, got %q", "test", got.Name())
	}
}

// TestProviderFactoryNoAvailable verifies that Default() returns nil when no
// provider is available.
func TestProviderFactoryNoAvailable(t *testing.T) {
	f := &ProviderFactory{
		providers: make(map[string]AIProvider),
	}
	got := f.Default()
	if got != nil {
		t.Errorf("expected nil default with no providers, got %v", got)
	}
}

// TestProviderFactoryFallback verifies that Default() falls back to a
// different available provider when the preferred one is unavailable.
func TestProviderFactoryFallback(t *testing.T) {
	f := &ProviderFactory{
		providers: make(map[string]AIProvider),
	}
	unavailable := &mockProvider{name: "unavail", available: false}
	available := &mockProvider{name: "fallback", available: true}
	f.Register(unavailable)
	f.Register(available)
	f.preferred = "unavail"

	got := f.Default()
	if got == nil {
		t.Fatal("expected non-nil fallback provider")
	}
	if got.Name() != "fallback" {
		t.Errorf("expected fallback provider, got %q", got.Name())
	}
}

// TestProviderFactoryAvailable verifies that Available() filters by IsAvailable().
func TestProviderFactoryAvailable(t *testing.T) {
	f := &ProviderFactory{
		providers: make(map[string]AIProvider),
	}
	f.Register(&mockProvider{name: "up", available: true})
	f.Register(&mockProvider{name: "down", available: false})

	avail := f.Available()
	if len(avail) != 1 {
		t.Fatalf("expected 1 available provider, got %d", len(avail))
	}
	if avail[0].Name() != "up" {
		t.Errorf("expected provider %q, got %q", "up", avail[0].Name())
	}
}

// TestProviderFactoryGet verifies Get() returns the registered provider.
func TestProviderFactoryGet(t *testing.T) {
	f := &ProviderFactory{
		providers: make(map[string]AIProvider),
	}
	mock := &mockProvider{name: "myp", available: true}
	f.Register(mock)

	got, ok := f.Get("myp")
	if !ok {
		t.Fatal("expected Get to find registered provider")
	}
	if got != mock {
		t.Error("expected exact same provider instance")
	}

	_, ok = f.Get("nonexistent")
	if ok {
		t.Error("expected Get to return false for unknown provider")
	}
}
