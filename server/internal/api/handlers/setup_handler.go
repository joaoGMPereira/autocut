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

	// Use deterministic job_id so the stream subscriber can connect before posting.
	jobID := "install-" + name
	go func() {
		logCh := make(chan string, 64)
		defer close(logCh)
		go func() {
			for line := range logCh {
				h.hub.Publish(jobID, hub.SSEEvent{Type: "log", Data: map[string]string{"message": line}})
			}
		}()
		if err := h.cfg.Install(context.Background(), name, logCh); err != nil {
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]string{"tool": name, "success": "true"}})
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

// GetInstallStream streams SSE events for an in-progress install job.
// The client subscribes by job_id returned from POST /api/setup/install/{tool}.
// Per the contract, the client may also subscribe before posting by using
// GET /api/setup/install/{tool}/stream — the hub will queue events once posted.
func (h *SetupHandler) GetInstallStream(w http.ResponseWriter, r *http.Request) {
	tool := r.PathValue("tool")
	if _, ok := h.cfg.Get(tool); !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "tool_not_found", "message": "tool not registered: " + tool})
		return
	}
	// The job_id for installs is derived from the tool name so the frontend can
	// open the stream before triggering the install POST.
	jobID := "install-" + tool
	h.hub.ServeSSE(w, r, jobID)
}

// GetCheckUpdate returns update availability for the named tool.
func (h *SetupHandler) GetCheckUpdate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("tool")
	info, err := h.cfg.CheckUpdate(r.Context(), name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "tool_not_found", "message": "tool not registered: " + name})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// GetWhisperModels returns the list of known Whisper model variants with download status.
func (h *SetupHandler) GetWhisperModels(w http.ResponseWriter, _ *http.Request) {
	models := h.cfg.WhisperModelStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"models": models})
}

// PostWhisperModel starts a background download of the named Whisper model and returns a job_id.
func (h *SetupHandler) PostWhisperModel(w http.ResponseWriter, r *http.Request) {
	model := r.PathValue("model")

	jobID := "whisper-" + model
	go func() {
		logCh := make(chan string, 64)
		defer close(logCh)
		go func() {
			for line := range logCh {
				h.hub.Publish(jobID, hub.SSEEvent{Type: "log", Data: map[string]string{"message": line}})
			}
		}()
		if err := h.cfg.DownloadWhisperModel(context.Background(), model, logCh); err != nil {
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]string{"message": err.Error()}})
			return
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]string{"model": model}})
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}

// GetWhisperModelStream streams SSE events for a Whisper model download job.
func (h *SetupHandler) GetWhisperModelStream(w http.ResponseWriter, r *http.Request) {
	model := r.PathValue("model")
	jobID := "whisper-" + model
	h.hub.ServeSSE(w, r, jobID)
}
