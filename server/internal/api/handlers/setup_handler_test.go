package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockConfigurator implements ConfiguratorFace for unit tests.
type mockConfigurator struct {
	tools []configurator.ToolStatus
	dir   *configurator.AutoCutDir
}

func (m *mockConfigurator) Status() []configurator.ToolStatus {
	return m.tools
}

func (m *mockConfigurator) Get(name string) (configurator.ToolValidator, bool) {
	for _, t := range m.tools {
		if t.Name == name {
			return &mockValidator{status: t}, true
		}
	}
	return nil, false
}

func (m *mockConfigurator) Install(_ context.Context, _ string, logCh chan<- string) error {
	logCh <- "mock install started"
	logCh <- "mock install done"
	return nil
}

func (m *mockConfigurator) Dir() *configurator.AutoCutDir {
	return m.dir
}

// mockValidator satisfies configurator.ToolValidator.
type mockValidator struct {
	status configurator.ToolStatus
}

func (v *mockValidator) Name() string                                     { return v.status.Name }
func (v *mockValidator) IsInstalled() bool                                { return v.status.Installed }
func (v *mockValidator) ResolvedPath() string                             { return v.status.Path }
func (v *mockValidator) Install(_ context.Context, _ chan<- string) error { return nil }
func (v *mockValidator) Instructions() string                             { return "" }
func (v *mockValidator) Status() configurator.ToolStatus                  { return v.status }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestSetupHandler(cfg ConfiguratorFace) *SetupHandler {
	h := hub.New()
	return NewSetupHandler(h, cfg)
}

func newMockCfg(tools ...configurator.ToolStatus) *mockConfigurator {
	// Build a temporary dir using /tmp — no filesystem side-effects in tests.
	dir := &configurator.AutoCutDir{
		Root:          "/tmp/.autocut-test",
		BinDir:        "/tmp/.autocut-test/bin",
		ModelsDir:     "/tmp/.autocut-test/models",
		TokensDir:     "/tmp/.autocut-test/tokens",
		CacheDir:      "/tmp/.autocut-test/cache",
		DownloadsDir:  "/tmp/.autocut-test/downloads",
		ThumbnailsDir: "/tmp/.autocut-test/thumbnails",
	}
	return &mockConfigurator{tools: tools, dir: dir}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestGetStatusReturns200(t *testing.T) {
	cfg := newMockCfg(
		configurator.ToolStatus{Name: "yt-dlp", Installed: true, Path: "/usr/bin/yt-dlp", Required: true},
		configurator.ToolStatus{Name: "ffmpeg", Installed: false, Required: true},
	)
	handler := newTestSetupHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	rr := httptest.NewRecorder()
	handler.GetStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var body struct {
		Tools []configurator.ToolStatus `json:"tools"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(body.Tools))
	}
}

func TestPostInstallReturnsJobID(t *testing.T) {
	cfg := newMockCfg(
		configurator.ToolStatus{Name: "yt-dlp", Installed: false, Required: true},
	)
	handler := newTestSetupHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/setup/install/yt-dlp", nil)
	req.SetPathValue("tool", "yt-dlp")
	rr := httptest.NewRecorder()
	handler.PostInstall(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	jobID, ok := body["job_id"]
	if !ok || jobID == "" {
		t.Fatalf("expected non-empty job_id, got %q", jobID)
	}
}

func TestPostInstallUnknownToolReturns404(t *testing.T) {
	cfg := newMockCfg(
		configurator.ToolStatus{Name: "yt-dlp", Installed: false, Required: true},
	)
	handler := newTestSetupHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/setup/install/nonexistent", nil)
	req.SetPathValue("tool", "nonexistent")
	rr := httptest.NewRecorder()
	handler.PostInstall(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if code := body["code"]; code != "tool_not_found" {
		t.Fatalf("expected code tool_not_found, got %q", code)
	}
}

func TestGetSetupDir(t *testing.T) {
	cfg := newMockCfg()
	handler := newTestSetupHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/setup/dir", nil)
	rr := httptest.NewRecorder()
	handler.GetDir(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	root, ok := body["root"]
	if !ok || root == "" {
		t.Fatalf("expected non-empty root, got %q", root)
	}
	if !strings.Contains(root, "autocut") {
		t.Fatalf("expected root to contain 'autocut', got %q", root)
	}
	// Verify all expected keys are present.
	for _, key := range []string{"bin_dir", "models_dir", "tokens_dir", "cache_dir", "downloads_dir", "thumbnails_dir"} {
		if v := body[key]; v == "" {
			t.Errorf("expected non-empty %q in response", key)
		}
	}
}
