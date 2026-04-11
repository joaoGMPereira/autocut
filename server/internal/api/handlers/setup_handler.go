package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// ConfiguratorFace is the interface the SetupHandler depends on.
// It is satisfied by *configurator.Configurator.
type ConfiguratorFace interface {
	Status() []configurator.ToolStatus
	Get(name string) (configurator.ToolValidator, bool)
	Install(ctx context.Context, name string, logCh chan<- string) error
	Dir() *configurator.AutoCutDir
	CheckUpdate(ctx context.Context, name string) (configurator.UpdateInfo, error)
	WhisperModelStatus() []configurator.WhisperModelInfo
	DownloadWhisperModel(ctx context.Context, name string, logCh chan<- string) error
}

// SetupHandler handles tool setup and status requests.
type SetupHandler struct {
	hub *hub.SSEHub
	cfg ConfiguratorFace
}

// NewSetupHandler creates a SetupHandler.
func NewSetupHandler(h *hub.SSEHub, cfg ConfiguratorFace) *SetupHandler {
	return &SetupHandler{hub: h, cfg: cfg}
}

// GetStatus returns the install status of all registered tools.
func (h *SetupHandler) GetStatus(w http.ResponseWriter, _ *http.Request) {
	tools := h.cfg.Status()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
}

// PostInstall starts a background install for the named tool and returns a job_id.
func (h *SetupHandler) PostInstall(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("tool")
	if _, ok := h.cfg.Get(name); !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "tool_not_found", "message": "tool not registered: " + name})
		return
	}

	jobID := newJobID()
	go func() {
		logCh := make(chan string, 64)
		defer close(logCh)
		go func() {
			for line := range logCh {
				h.hub.Publish(jobID, hub.SSEEvent{Type: "log", Data: map[string]string{"message": line}})
			}
		}()
		if err := h.cfg.Install(r.Context(), name, logCh); err != nil {
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]string{"tool": name}})
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}

// GetDir returns the AutoCut directory paths.
func (h *SetupHandler) GetDir(w http.ResponseWriter, _ *http.Request) {
	dir := h.cfg.Dir()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"root":           dir.Root,
		"bin_dir":        dir.BinDir,
		"models_dir":     dir.ModelsDir,
		"tokens_dir":     dir.TokensDir,
		"cache_dir":      dir.CacheDir,
		"downloads_dir":  dir.DownloadsDir,
		"thumbnails_dir": dir.ThumbnailsDir,
	})
}
