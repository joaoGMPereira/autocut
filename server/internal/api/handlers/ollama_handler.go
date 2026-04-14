package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// OllamaHandler proxies requests to the local Ollama API.
type OllamaHandler struct {
	ollamaURL  string
	httpClient *http.Client
}

// NewOllamaHandler creates an OllamaHandler targeting Ollama at the default local URL.
func NewOllamaHandler() *OllamaHandler {
	return &OllamaHandler{
		ollamaURL: "http://localhost:11434",
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// ollamaTagsResponse is the shape returned by Ollama's GET /api/tags.
type ollamaTagsResponse struct {
	Models []ollamaTagModel `json:"models"`
}

type ollamaTagModel struct {
	Name string `json:"name"`
}

// ollamaModel is the shape returned by GET /api/ollama/models.
type ollamaModel struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// ollamaStatusResponse is the full response from GET /api/ollama/models.
type ollamaStatusResponse struct {
	Available bool          `json:"available"`
	Models    []ollamaModel `json:"models"`
}

// GetOllamaModels queries the local Ollama API for available models.
// Returns {available: false, models: []} when Ollama is unreachable — never an HTTP error.
func (h *OllamaHandler) GetOllamaModels(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := h.httpClient.Get(h.ollamaURL + "/api/tags") //nolint:noctx
	if err != nil {
		slog.Debug("ollama unreachable", "err", err)
		_ = json.NewEncoder(w).Encode(ollamaStatusResponse{Available: false, Models: []ollamaModel{}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		slog.Warn("ollama returned error status", "status", resp.StatusCode)
		_ = json.NewEncoder(w).Encode(ollamaStatusResponse{Available: false, Models: []ollamaModel{}})
		return
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		slog.Error("failed to decode ollama tags", "err", err)
		_ = json.NewEncoder(w).Encode(ollamaStatusResponse{Available: false, Models: []ollamaModel{}})
		return
	}

	models := make([]ollamaModel, len(tags.Models))
	for i, m := range tags.Models {
		models[i] = ollamaModel{Name: m.Name}
	}

	slog.Info("ollama models fetched", "count", len(models))
	_ = json.NewEncoder(w).Encode(ollamaStatusResponse{Available: true, Models: models})
}
