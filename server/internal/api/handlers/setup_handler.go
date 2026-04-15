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
	Refresh() []configurator.ToolStatus
	Sync() []configurator.ToolStatus
	Get(name string) (configurator.ToolValidator, bool)
	Install(ctx context.Context, name string, logCh chan<- string) error
	Dir() *configurator.AutoCutDir
	CheckUpdate(ctx context.Context, name string) (configurator.UpdateInfo, error)
	WhisperModelStatus() []configurator.WhisperModelInfo
	DownloadWhisperModel(ctx context.Context, name string, logCh chan<- string) error
	Hardware() (configurator.HardwareProfile, configurator.ModelRecommendation)
	SetCustomPath(ctx context.Context, toolName, path string) error
	ClearCustomPath(ctx context.Context, toolName string) error
	ApplyCustomPath(ctx context.Context, toolName, customPath string) (configurator.ToolStatus, string, error)
	DetectAndClearCustomPath(ctx context.Context, toolName string) (configurator.ToolStatus, error)
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

// GetHardware returns hardware profile and model recommendations.
func (h *SetupHandler) GetHardware(w http.ResponseWriter, _ *http.Request) {
	hw, rec := h.cfg.Hardware()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"hardware":       hw,
		"recommendation": rec,
	})
}

// GetWhisperModelStream streams SSE events for a Whisper model download job.
func (h *SetupHandler) GetWhisperModelStream(w http.ResponseWriter, r *http.Request) {
	model := r.PathValue("model")
	jobID := "whisper-" + model
	h.hub.ServeSSE(w, r, jobID)
}

// PostCustomPath sets a custom binary path for the named tool.
// POST /api/setup/tools/{tool}/path
// Body: {"path": "/absolute/path/to/binary"}
// 200: {"tool": ToolStatus}
// 422: {"code": "...", "message": "..."}
func (h *SetupHandler) PostCustomPath(w http.ResponseWriter, r *http.Request) {
	tool := r.PathValue("tool")
	w.Header().Set("Content-Type", "application/json")

	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "invalid_path",
			"message": "request body must be {\"path\": \"/absolute/path\"}",
		})
		return
	}

	status, errCode, err := h.cfg.ApplyCustomPath(r.Context(), tool, body.Path)
	if err != nil {
		if errCode == "tool_not_found" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    errCode,
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tool": status})
}

// PostSync copies binaries from ~/.autocut/bin into the current dir (useful in dev),
// re-discovers all tools, and returns fresh statuses.
func (h *SetupHandler) PostSync(w http.ResponseWriter, _ *http.Request) {
	tools := h.cfg.Sync()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
}

// DeleteCustomPath clears the custom binary path for the named tool.
// DELETE /api/setup/tools/{tool}/path
// 200: {"tool": ToolStatus}
func (h *SetupHandler) DeleteCustomPath(w http.ResponseWriter, r *http.Request) {
	tool := r.PathValue("tool")
	w.Header().Set("Content-Type", "application/json")

	status, err := h.cfg.DetectAndClearCustomPath(r.Context(), tool)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "tool_not_found",
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tool": status})
}
