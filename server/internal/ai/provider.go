package ai

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// AIProvider is the abstraction over any AI backend that can generate text.
// Kotlin ref: AiProvider sealed class — Go uses an interface instead of sealed.
type AIProvider interface {
	Name() string
	IsAvailable() bool
	GenerateSync(ctx context.Context, req GenerateRequest) (string, error)
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan string, error)
}

// OllamaProvider adapts OllamaClient to the AIProvider interface.
// Kotlin ref: OllamaAiProvider implements AiProvider
type OllamaProvider struct {
	client     *OllamaClient
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

// NewOllamaProvider creates an OllamaProvider pointing at baseURL.
// timeout applies to both availability checks and generate calls.
func NewOllamaProvider(baseURL string, timeout time.Duration) *OllamaProvider {
	client := New(baseURL, timeout)
	return &OllamaProvider{
		client:  client,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		log: slog.With("component", "ollama_provider"),
	}
}

// Name returns the provider identifier.
func (p *OllamaProvider) Name() string { return "ollama" }

// IsAvailable performs a quick GET to the Ollama root URL.
// Returns true when the status code is < 500 (server is up, even if 404).
func (p *OllamaProvider) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/", nil)
	if err != nil {
		p.log.Debug("IsAvailable: build request failed", "err", err)
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.log.Debug("IsAvailable: request failed", "err", err)
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode < 500
	p.log.Debug("IsAvailable", "status", resp.StatusCode, "available", available)
	return available
}

// GenerateSync delegates to the underlying OllamaClient.
func (p *OllamaProvider) GenerateSync(ctx context.Context, req GenerateRequest) (string, error) {
	return p.client.GenerateSync(ctx, req)
}

// GenerateStream delegates to the underlying OllamaClient.
func (p *OllamaProvider) GenerateStream(ctx context.Context, req GenerateRequest) (<-chan string, error) {
	return p.client.GenerateStream(ctx, req)
}

// ---------------------------------------------------------------------------
// ProviderFactory
// ---------------------------------------------------------------------------

// ProviderFactory holds registered AI providers and routes requests to the
// best available one.
// Kotlin ref: AiProviderFactory — Go version is registry-based instead of
// sealed-class dispatch.
type ProviderFactory struct {
	providers map[string]AIProvider
	order     []string // registration order for deterministic fallback
	preferred string
	log       *slog.Logger
}

// NewProviderFactory creates a ProviderFactory pre-registered with an
// OllamaProvider at ollamaURL and sets it as the preferred provider.
func NewProviderFactory(ollamaURL string) *ProviderFactory {
	f := &ProviderFactory{
		providers: make(map[string]AIProvider),
		log:       slog.With("component", "provider_factory"),
	}
	p := NewOllamaProvider(ollamaURL, 30*time.Second)
	f.Register(p)
	f.preferred = p.Name()
	return f
}

// Register adds a provider to the factory, indexed by Name().
// Replaces any previously registered provider with the same name.
func (f *ProviderFactory) Register(p AIProvider) {
	if _, exists := f.providers[p.Name()]; !exists {
		f.order = append(f.order, p.Name())
	}
	f.providers[p.Name()] = p
	f.logger().Debug("provider registered", "name", p.Name())
}

// Get returns the named provider and whether it was found.
func (f *ProviderFactory) Get(name string) (AIProvider, bool) {
	p, ok := f.providers[name]
	return p, ok
}

// SetPreferred sets the name of the preferred provider.
// The name must already be registered; if not, the call is silently ignored.
func (f *ProviderFactory) SetPreferred(name string) {
	if _, ok := f.providers[name]; !ok {
		f.logger().Warn("SetPreferred: unknown provider", "name", name)
		return
	}
	f.preferred = name
}

// Default returns the preferred provider when it is available, falls back to
// the first available provider in registration order, or nil if none is available.
func (f *ProviderFactory) Default() AIProvider {
	if p, ok := f.providers[f.preferred]; ok && p.IsAvailable() {
		return p
	}
	for _, name := range f.order {
		if p := f.providers[name]; p.IsAvailable() {
			return p
		}
	}
	f.logger().Warn("Default: no available provider")
	return nil
}

// Available returns all providers for which IsAvailable() is true,
// in registration order for deterministic results.
func (f *ProviderFactory) Available() []AIProvider {
	var result []AIProvider
	for _, name := range f.order {
		if p := f.providers[name]; p.IsAvailable() {
			result = append(result, p)
		}
	}
	return result
}

// GenerateSync is a convenience method that routes to the default provider.
// Returns an error when no provider is available.
func (f *ProviderFactory) GenerateSync(ctx context.Context, req GenerateRequest) (string, error) {
	p := f.Default()
	if p == nil {
		return "", fmt.Errorf("provider_factory: no available AI provider")
	}
	return p.GenerateSync(ctx, req)
}

// logger returns f.log if set, or the slog default logger.
// This allows ProviderFactory to be constructed in tests without NewProviderFactory.
func (f *ProviderFactory) logger() *slog.Logger {
	if f.log != nil {
		return f.log
	}
	return slog.Default()
}
