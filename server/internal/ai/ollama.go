package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ollamaGenerateResponse is the per-line JSON shape from Ollama's NDJSON stream.
// Kotlin ref: OllamaClient.generate() — parses response.jsonObject["response"]
type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// OllamaClient is a thin HTTP client for Ollama's /api/generate endpoint.
// Kotlin ref: OllamaClient (Go version uses net/http instead of ProcessRunner+curl)
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

// New creates an OllamaClient pointing at baseURL with the given HTTP timeout.
// baseURL should be the Ollama root URL, e.g. "http://localhost:11434".
func New(baseURL string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: slog.With("component", "ollama"),
	}
}

// GenerateStream sends a streaming generate request to Ollama and returns a
// channel that receives individual response tokens.
//
// The caller is responsible for draining the channel or cancelling ctx.
// The channel is closed when Ollama signals done=true or when an error occurs.
//
// Kotlin ref: OllamaClient.generate() — adds true streaming; Kotlin always
// used stream=false and returned the full string in one shot.
func (c *OllamaClient) GenerateStream(ctx context.Context, req GenerateRequest) (<-chan string, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: do request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	ch := make(chan string, 32)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var chunk ollamaGenerateResponse
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				c.log.Error("ollama: unmarshal chunk", "err", err, "line", line)
				return
			}

			if chunk.Response != "" {
				select {
				case ch <- chunk.Response:
				case <-ctx.Done():
					return
				}
			}

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if ctx.Err() == nil {
				c.log.Error("ollama: scanner error", "err", err)
			}
		}
	}()

	return ch, nil
}

// GenerateSync calls GenerateStream and concatenates all tokens into a single
// string. It blocks until the stream is complete or ctx is cancelled.
//
// Kotlin ref: OllamaClient.generate() — this is the equivalent synchronous path.
func (c *OllamaClient) GenerateSync(ctx context.Context, req GenerateRequest) (string, error) {
	ch, err := c.GenerateStream(ctx, req)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for token := range ch {
		sb.WriteString(token)
	}

	if ctx.Err() != nil {
		return "", fmt.Errorf("ollama: generate: %w", ctx.Err())
	}

	return sb.String(), nil
}
