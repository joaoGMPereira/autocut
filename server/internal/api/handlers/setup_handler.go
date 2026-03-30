package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// ConfiguratorFace is the interface SetupHandler depends on.
// Using an interface instead of *configurator.Configurator enables mock-based testing.
type ConfiguratorFace interface {
	Status() []configurator.ToolStatus
	Get(name string) (configurator.ToolValidator, bool)
	Install(ctx context.Context, name string, logCh chan<- string) error
	Dir() *configurator.AutoCutDir
}

// SetupHandler handles tool setup and directory inspection routes.
type SetupHandler struct {
	hub *hub.SSEHub
	cfg ConfiguratorFace
}

// NewSetupHandler creates a SetupHandler.
func NewSetupHandler(hub *hub.SSEHub, cfg ConfiguratorFace) *SetupHandler {
	return &SetupHandler{hub: hub, cfg: cfg}
}

// GetStatus handles GET /api/setup/status.
// Response: {"tools": [{"name":"ffmpeg","installed":true,"path":"/usr/local/bin/ffmpeg","required":true}, ...]}
func (h *SetupHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	tools := h.cfg.Status()
	writeJSON(w, http.StatusOK, map[string]any{"tools": tools})
}

// PostInstall handles POST /api/setup/install/{tool}.
// Returns 202 with a job_id; progress is streamed via SSE on /api/setup/install/{tool}/stream.
func (h *SetupHandler) PostInstall(w http.ResponseWriter, r *http.Request) {
	toolName := r.PathValue("tool")
	if _, ok := h.cfg.Get(toolName); !ok {
		writeError(w, http.StatusNotFound, "tool_not_found", "tool not registered: "+toolName)
		return
	}

	jobID := "setup-install-" + toolName

	go func() {
		logCh := make(chan string, 32)
		ctx := context.Background()

		// Forward log messages as SSE events.
		go func() {
			for msg := range logCh {
				h.hub.Publish(jobID, hub.SSEEvent{
					Type: "log",
					Data: map[string]any{"message": msg},
				})
			}
		}()

		err := h.cfg.Install(ctx, toolName, logCh)
		close(logCh)

		if err != nil {
			slog.Error("tool install failed", "component", "api", "handler", "setup",
				"tool", toolName, "err", err)
			h.hub.Publish(jobID, hub.SSEEvent{
				Type: "error",
				Data: map[string]any{"message": err.Error()},
			})
			return
		}

		h.hub.Publish(jobID, hub.SSEEvent{
			Type: "done",
			Data: map[string]any{"success": true},
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

// GetInstallStream handles GET /api/setup/install/{tool}/stream.
// Streams SSE events for the install job identified by the tool name.
func (h *SetupHandler) GetInstallStream(w http.ResponseWriter, r *http.Request) {
	toolName := r.PathValue("tool")
	h.hub.ServeSSE(w, r, "setup-install-"+toolName)
}

// GetDir handles GET /api/setup/dir.
// Returns all AutoCut directory paths.
func (h *SetupHandler) GetDir(w http.ResponseWriter, r *http.Request) {
	d := h.cfg.Dir()
	writeJSON(w, http.StatusOK, map[string]string{
		"root":          d.Root,
		"bin_dir":       d.BinDir,
		"models_dir":    d.ModelsDir,
		"tokens_dir":    d.TokensDir,
		"cache_dir":     d.CacheDir,
		"downloads_dir": d.DownloadsDir,
		"thumbnails_dir": d.ThumbnailsDir,
	})
}
